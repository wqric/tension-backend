package internal

import (
	"errors"
	"math/rand"

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

func AutoAssignWorkouts(userID int, months int, freq int) ([]UserWorkoutResponse, error) {
	var user User
	var suitableWorkouts []Workout
	response := make([]UserWorkoutResponse, 0)

	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	// 1. Загружаем подходящие тренировки
	err := db.Preload("Exercises").
		Where("aim = ? AND difficult = ?", user.Aim, user.Difficult).
		Find(&suitableWorkouts).Error

	if err != nil || len(suitableWorkouts) == 0 {
		return nil, errors.New("no workouts found for new parameters")
	}

	// --- ДОБАВЛЯЕМ ПЕРЕМЕШИВАНИЕ ---
	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())
	// Перемешиваем исходный список тренировок, чтобы порядок всегда был разным
	rand.Shuffle(len(suitableWorkouts), func(i, j int) {
		suitableWorkouts[i], suitableWorkouts[j] = suitableWorkouts[j], suitableWorkouts[i]
	})
	// ------------------------------

	totalWorkoutsToGen := freq * 4 * months
	daysStep := 7.0 / float64(freq)

	err = db.Transaction(func(tx *gorm.DB) error {
		// Удаляем только будущие невыполненные
		tx.Where("user_id = ? AND is_done = ? AND scheduled_date > ?",
			userID, false, time.Now()).Delete(&UserWorkout{})

		startDate := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)

		for i := 0; i < totalWorkoutsToGen; i++ {
			// Выбираем случайную тренировку из списка подходящих
			// Используем rand.Intn, чтобы выбор был непредсказуемым
			randomIndex := rand.Intn(len(suitableWorkouts))
			workout := suitableWorkouts[randomIndex]

			daysOffset := int(float64(i) * daysStep)
			scheduledDate := startDate.AddDate(0, 0, daysOffset)

			userWorkout := UserWorkout{
				UserID:        user.ID,
				WorkoutID:     workout.ID,
				ScheduledDate: scheduledDate,
				IsDone:        false,
			}

			if err := tx.Create(&userWorkout).Error; err != nil {
				return err
			}

			response = append(response, UserWorkoutResponse{
				WorkoutID:     workout.ID,
				Title:         workout.Title,
				Desc:          workout.Desc, // Не забудь добавить описание
				ScheduledDate: scheduledDate,
				IsDone:        false,
				Exercises:     workout.Exercises,
			})
		}
		return nil
	})

	return response, err
}

func GetUserStats(userID int) (UserStatsResponse, error) {
	var stats UserStatsResponse
	var totalAssigned int64
	var totalDone int64
	db.Model(&UserWorkout{}).Where("user_id = ?", userID).Count(&totalAssigned)
	db.Model(&UserWorkout{}).Where("user_id = ? AND is_done = ?", userID, true).Count(&totalDone)

	stats.TotalWorkouts = int(totalDone)

	if totalAssigned > 0 {
		rate := (float64(totalDone) / float64(totalAssigned)) * 100
		stats.CompletionRate = math.Round(rate*10) / 10
	}

	var completedWorkouts []UserWorkout
	db.Preload("Workout.Exercises").
		Where("user_id = ? AND is_done = ?", userID, true).
		Find(&completedWorkouts)

	exCount := 0
	for _, uw := range completedWorkouts {
		exCount += len(uw.Workout.Exercises)
	}
	stats.TotalExercises = exCount

	return stats, nil
}

func CompleteWorkout(userID int, workoutID int, date string) error {
	result := db.Model(&UserWorkout{}).
		Where("user_id = ? AND workout_id = ? AND DATE(scheduled_date) = DATE(?)",
			userID, workoutID, date).
		Update("is_done", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("Запись не найдена: User %d, Workout %d, Date %s", userID, workoutID, date)
		return errors.New("тренировка не найдена для указанного пользователя и даты")
	}

	return nil
}

func UpdateUser(id int, data map[string]interface{}) (*User, error) {
	var user User

	if err := db.First(&user, id).Error; err != nil {
		return nil, err
	}

	if err := db.Model(&user).Updates(data).Error; err != nil {
		return nil, err
	}

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
