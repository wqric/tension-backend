package internal

import (
	"github.com/gin-gonic/gin"
)

func NewUser(c *gin.Context) {
	var rq RegistrationRQ

	// Привязываем JSON из тела запроса к структуре
	if err := c.ShouldBindJSON(&rq); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	// Валидация обязательных полей
	if rq.Email == "" || rq.Password == "" {
		c.JSON(400, gin.H{
			"error": "All fields are required: email, password, name, lastname",
		})
		return
	}

	// Создаем пользователя
	user, err := CreateUser(rq.Email, rq.Password)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "internal error",
		})
	}
	token, err := GenerateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "internal error",
		})
	}
	user.Token = token
	// Возвращаем успешный ответ (исключая пароль из ответа)
	c.JSON(200, user)
}

func GetStats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := GetUserStats(userID.(int))
	if err != nil {
		c.JSON(500, gin.H{"error": "Could not fetch stats", "details": err.Error()})
		return
	}

	c.JSON(200, stats)
}

func Login(c *gin.Context) {
	var rq LoginRQ
	if err := c.ShouldBindJSON(&rq); err != nil {
		c.JSON(400, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}
	if rq.Email == "" || rq.Password == "" {
		c.JSON(400, gin.H{
			"error": "All fields are required: email, password",
		})
		return
	}
	user := GetUserProfileByCredentials(rq.Email, rq.Password)
	if user == nil {
		c.JSON(401, gin.H{
			"error": "Invalid credentials",
		})
		return
	}
	token, err := GenerateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "internal error",
		})
	}
	user.Token = token
	c.JSON(200, user)
}

func GetProfile(c *gin.Context) {
	userID := c.GetInt("userID")
	user := GetUserProfile(userID)
	if user == nil {
		c.JSON(404, gin.H{
			"error": "user not found",
		})
		return
	}
	token, err := GenerateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "internal error",
		})
		return
	}
	user.Token = token
	c.JSON(200, user)
}

func SetupWorkoutPlan(c *gin.Context) {
	// 1. Извлекаем userID, который положил туда AuthMiddleware
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	schedule, err := AutoAssignWorkouts(userID.(int))
	if err != nil {
		c.JSON(500, gin.H{
			"error":   "Failed to generate workout plan",
			"details": err.Error(),
		})
		return
	}

	// 3. Если тренировок не нашлось под такие параметры
	if len(schedule) == 0 {
		c.JSON(404, gin.H{"error": "No suitable workouts found for your level/aim"})
		return
	}

	// 4. Возвращаем успешный ответ со списком
	c.JSON(200, gin.H{
		"status":   "plan_created",
		"workouts": schedule,
	})
}

func GetWorkouts(c *gin.Context) {
	userID, _ := c.Get("userID")
	var userWorkouts []UserWorkout
	var response []UserWorkoutResponse

	// 1. Загружаем связи + тренировки + упражнения через двойной Preload
	err := db.Where("user_id = ?", userID).
		Preload("Workout").
		Preload("Workout.Exercises").
		Find(&userWorkouts).Error

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 2. Перекладываем данные в структуру ответа (DTO)
	for _, uw := range userWorkouts {
		response = append(response, UserWorkoutResponse{
			WorkoutID:     uw.WorkoutID,
			Title:         uw.Workout.Title, // Берем из вложенной структуры Workout
			Desc:          uw.Workout.Desc,
			ScheduledDate: uw.ScheduledDate,
			IsDone:        uw.IsDone,
			Exercises:     uw.Workout.Exercises,
		})
	}

	c.JSON(200, response)
}

func MarkWorkoutDone(c *gin.Context) {
	userID, _ := c.Get("userID")

	// Ожидаем JSON с ID тренировки и датой, на которую она была назначена
	var rq struct {
		WorkoutID int    `json:"workout_id"`
		Date      string `json:"date"` // Формат "2026-02-07"
	}

	if err := c.ShouldBindJSON(&rq); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// Вызываем функцию обновления в базе
	err := CompleteWorkout(userID.(int), rq.WorkoutID, rq.Date)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to update status", "details": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status":     "success",
		"message":    "workout marked as completed",
		"workout_id": rq.WorkoutID,
	})
}

func UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Читаем JSON сразу в map[string]interface{}
	// Это позволит передавать любое количество полей (только вес, или только цель, или всё вместе)
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	updatedUser, err := UpdateUser(userID.(int), data)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update user", "details": err.Error()})
		return
	}

	// 4. Возвращаем обновленного пользователя (GORM уже обновил объект в памяти)
	c.JSON(200, gin.H{
		"status": "success",
		"user":   updatedUser,
	})
}
