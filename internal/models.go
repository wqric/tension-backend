package internal

import "time"

type User struct {
	ID        int       `gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name"`
	Lastname  string    `json:"lastname"`
	Email     string    `json:"email" gorm:"unique"`
	Password  string    `json:"-"`
	Weight    float32   `json:"weight"`
	Height    float32   `json:"height"`
	Aim       int       `json:"aim"`
	Difficult int       `json:"difficult"`
	Token     string    `json:"token,omitempty" gorm:"-"`
	Workouts  []Workout `gorm:"many2many:user_workouts;"`
}

type UserWorkout struct {
	UserID    int `gorm:"primaryKey"`
	WorkoutID int `gorm:"primaryKey"`
	// Добавь эту строку ниже:
	Workout       Workout   `gorm:"foreignKey:WorkoutID"`
	ScheduledDate time.Time `gorm:"primaryKey"`
	IsDone        bool      `gorm:"default:false"`
}

type Workout struct {
	ID        int         `gorm:"primaryKey;autoIncrement"`
	Title     string      `json:"title"`
	Desc      string      `json:"desc"`
	Difficult int         `json:"difficult"`
	Aim       int         `json:"aim"`
	Exercises []Exercises `json:"exercises" gorm:"many2many:workout_exercises;"`
}

type Exercises struct {
	ID    int    `gorm:"primaryKey;autoIncrement"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Sets  int    `json:"sets"`
	Reps  int    `json:"reps"`
	Rest  int    `json:"rest"`
}

type UserWorkoutResponse struct {
	WorkoutID     int         `json:"workout_id" gorm:"-"`
	Title         string      `json:"title" gorm:"-"`
	Desc          string      `json:"desc" gorm:"-"`
	ScheduledDate time.Time   `json:"scheduled_date" gorm:"-"`
	IsDone        bool        `json:"is_done" gorm:"-"`
	Exercises     []Exercises `json:"exercises" gorm:"-"` // GORM проигнорирует это поле
}

type RegistrationRQ struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRQ struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRQ struct {
	Name      string  `json:"name,omitempty"`
	Lastname  string  `json:"lastname,omitempty"`
	Weight    float32 `json:"weight,omitempty"`
	Height    float32 `json:"height,omitempty"`
	Aim       int     `json:"aim,omitempty"`
	Difficult int     `json:"difficult,omitempty"`
}

type UserStatsResponse struct {
	TotalWorkouts   int     `json:"total_workouts"`   // Всего выполненных
	CompletionRate  float64 `json:"completion_rate"`  // % выполненных от назначенных
	TotalExercises  int     `json:"total_exercises"`  // Сколько всего подходов/упражнений сделано
	CurrentStreak   int     `json:"current_streak"`   // Серия дней без пропусков
	FavoriteWorkout string  `json:"favorite_workout"` // Самая частая тренировка
}
