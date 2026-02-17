package internal

import (
	"errors"
	"log"
	"math"
	"time"

	"gorm.io/gorm"
)

func GetUserProfile(id int) *User {
	var user User
	if result := db.First(&user, id); result.Error != nil {
		return nil
	}
	return &user
}

func CreateUser(email, password string) (*User, error) {
	user := &User{
		Email:    email,
		Password: password,
	}
	result := db.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user, nil
}

func AutoAssignWorkouts(userID int) ([]UserWorkoutResponse, error) {
	var user User
	var suitableWorkouts []Workout
	response := make([]UserWorkoutResponse, 0)

	// 1. Получаем текущий профиль пользователя
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	// 2. Ищем новые подходящие тренировки
	err := db.Preload("Exercises").
		Where("aim = ? AND difficult = ?", user.Aim, user.Difficult).
		Limit(3).
		Find(&suitableWorkouts).Error

	if err != nil || len(suitableWorkouts) == 0 {
		return nil, errors.New("no workouts found for new parameters")
	}

	// 3. Транзакция для очистки и перезаписи
	err = db.Transaction(func(tx *gorm.DB) error {

		// --- ЛОГИКА УМНОГО УДАЛЕНИЯ ---
		// Удаляем только те UserWorkout, чьи параметры в таблице Workout
		err := tx.Where("user_id = ? AND is_done = ? AND workout_id IN (?)",
			userID,
			false,
			tx.Table("workouts").
				Select("id").
				Where("aim != ? OR difficult != ?", user.Aim, user.Difficult),
		).Delete(&UserWorkout{}).Error

		if err != nil {
			return err
		}
		// ------------------------------

		// Проверяем, нужно ли назначать новые (если после удаления пусто или план не полон)
		var count int64
		tx.Model(&UserWorkout{}).Where("user_id = ? AND is_done = ?", userID, false).Count(&count)

		if count < 3 {
			nextDate := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)

			for i := 0; i < int(3-count); i++ {
				workout := suitableWorkouts[i%len(suitableWorkouts)]

				userWorkout := UserWorkout{
					UserID:        user.ID,
					WorkoutID:     workout.ID,
					ScheduledDate: nextDate,
					IsDone:        false,
				}

				if err := tx.Create(&userWorkout).Error; err != nil {
					return err
				}

				// Добавляем в ответ (только новые)
				response = append(response, UserWorkoutResponse{
					WorkoutID:     workout.ID,
					Title:         workout.Title,
					ScheduledDate: nextDate,
					Exercises:     workout.Exercises,
				})
				nextDate = nextDate.AddDate(0, 0, 2)
			}
		}

		return nil
	})

	return response, err
}

func GetUserStats(userID int) (UserStatsResponse, error) {
	var stats UserStatsResponse
	var totalAssigned int64
	var totalDone int64

	// 1. Считаем тренировки
	db.Model(&UserWorkout{}).Where("user_id = ?", userID).Count(&totalAssigned)
	db.Model(&UserWorkout{}).Where("user_id = ? AND is_done = ?", userID, true).Count(&totalDone)

	stats.TotalWorkouts = int(totalDone)

	// 2. Округляем Rate (33.3333 -> 33.3)
	if totalAssigned > 0 {
		rate := (float64(totalDone) / float64(totalAssigned)) * 100
		stats.CompletionRate = math.Round(rate*10) / 10
	}

	// 3. Считаем упражнения (Вариант через циклы, если SQL падает)
	var completedWorkouts []UserWorkout
	db.Preload("Workout.Exercises").
		Where("user_id = ? AND is_done = ?", userID, true).
		Find(&completedWorkouts)

	exCount := 0
	for _, uw := range completedWorkouts {
		exCount += len(uw.Workout.Exercises)
	}
	stats.TotalExercises = exCount

	// 4. Любимая тренировка
	// ... твой код с FavoriteWorkout ...

	return stats, nil
}

func CompleteWorkout(userID int, workoutID int, date string) error {
	// Используем DATE(scheduled_date), чтобы сравнивать только календарный день
	result := db.Model(&UserWorkout{}).
		Where("user_id = ? AND workout_id = ? AND DATE(scheduled_date) = DATE(?)",
			userID, workoutID, date).
		Update("is_done", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		// Логируем для отладки, что именно мы искали
		log.Printf("Запись не найдена: User %d, Workout %d, Date %s", userID, workoutID, date)
		return errors.New("тренировка не найдена для указанного пользователя и даты")
	}

	return nil
}

func UpdateUser(id int, data map[string]interface{}) (*User, error) {
	var user User

	// 1. Сначала находим пользователя со всеми текущими данными
	if err := db.First(&user, id).Error; err != nil {
		return nil, err
	}

	// 2. Обновляем найденную запись переданными полями
	// Updates при работе со структурой (user) автоматически заполнит её новыми данными
	if err := db.Model(&user).Updates(data).Error; err != nil {
		return nil, err
	}

	// Теперь объект 'user' содержит и старые поля, которые мы не трогали,
	// и новые поля, которые пришли в 'data'
	return &user, nil
}

func GetUserProfileByCredentials(email, password string) *User {
	var user User
	if result := db.Where("email = ? AND password = ?", email, password).First(&user); result.Error != nil {
		return nil
	}
	return &user
}

func GetUserSchedule(userID int) ([]UserWorkoutResponse, error) {
	var results []UserWorkoutResponse

	// 1. Получаем основные данные тренировок и информацию о выполнении/дате
	err := db.Table("user_workouts").
		Select("user_workouts.workout_id, workouts.title, workouts.desc, workouts.difficult, workouts.aim, user_workouts.scheduled_date, user_workouts.is_done").
		Joins("JOIN workouts ON workouts.id = user_workouts.workout_id").
		Where("user_workouts.user_id = ?", userID).
		Order("user_workouts.scheduled_date ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 2. Для каждой тренировки подгружаем её упражнения
	for i := range results {
		var exercises []Exercises
		// Ищем упражнения через связующую таблицу workout_exercises
		err := db.Table("exercises").
			Joins("JOIN workout_exercises ON workout_exercises.exercises_id = exercises.id").
			Where("workout_exercises.workout_id = ?", results[i].WorkoutID).
			Find(&exercises).Error

		if err == nil {
			results[i].Exercises = exercises
		}
	}

	return results, nil
}
