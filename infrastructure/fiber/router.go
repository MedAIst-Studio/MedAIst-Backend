//	@title		MedAIist-Backend API
//	@version	1.0
//	@description
//	@description	## How to use the API
//	@description	1- Click which endpoint you want to use.
//	@description	2- Click "Try it out" button.
//	@description	3- Add your request body or your parameters which are showed and required by the endpoint.
//	@description	4- Click "Execute" button.
//	@description	5- You will see the response.
//	@description
//	@description	Some endpoints require authentication. In this case, you need to log in first.
//	@description	I created two types of users for this project: admin and regular user.
//	@description	Just send a POST request as below at [here](http://localhost:3000/swagger/index.html/).
//	@description	After login, you will get a JWE token in cookies.
//	@description	If you're using cookie-based auth, the cookie will be sent automatically.
//	@description	Alternatively, you can use Bearer Token authentication via the "Authorize" button.
//	@description
//	@description	### Login Request For Admin
//	@description	```json
//	@description	{
//	@description	"email": "admin@admin.com",
//	@description	"password": "admin123"
//	@description	}
//	@description	```
//	@description
//	@description	### Login Request For User
//	@description	```json
//	@description	{
//	@description	"email": "user@user.com",
//	@description	"password": "user1234"
//	@description	}
//	@description	```
//	@description
//	@description	## Error Handling
//	@description	All error responses will follow this JSON format:
//	@description
//	@description	```json
//	@description	{
//	@description	"message": string,
//	@description	"code": int
//	@description	}
//	@description	```
//	@description	### Example
//	@description	```json
//	@description	{
//	@description	"message": "invalid request",
//	@description	"code": 400
//	@description	}
//	@description	```
//	@description
//	@description	## Reminder
//	@description	I did not use `/api` prefix for the endpoint routes. Because I love to host my API on "api" subdomain.
//	@description	Status code with `2xx` is a success code.
//	@description	Status code with `4xx` is a client error code.
//	@description	Status code with `5xx` is a server error code.
//

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Enter your Bearer token in the format **Bearer &lt;token&gt;**
//
// @host						localhost:3000
package fiber

import (
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/admin"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/auth"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/chat"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/healthcheck"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/quiz"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/user"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"

	fiberSwagger "github.com/swaggo/fiber-swagger"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	dc := NewAuthDependencyContainer()

	middlewareManager := NewMiddlewareManager(dc.jweTokenService, dc.slogLogger)

	healthcheckHandler := healthcheck.NewHealthcheckHandler()

	if !domain.IsProdEnv() {
		domain.CookieSecure = false
	}

	signupHandler := auth.NewSignupHandler(&auth.SignupConfig{
		Repo:         dc.Repo,
		TokenService: dc.jweTokenService,
		EmailService: dc.mailerSender,
		Validator:    dc.Validator,
		Logger:       dc.slogLogger,
	})

	loginHandler := auth.NewLoginHandler(&auth.LoginConfig{
		Repo:         dc.Repo,
		TokenService: dc.jweTokenService,
		Validator:    dc.Validator,
		Logger:       dc.slogLogger,
	})

	exchangeHandler := auth.NewExchangeHandler(dc.inMemoryClient)

	googleOAuthService := NewGoogleOAuthService(dc)
	facebookOAuthService := NewFacebookOAuthService(dc)

	logoutHandler := auth.NewLogoutHandler(dc.Repo)
	refreshTokenHandler := auth.NewRefreshTokenHandler(dc.Repo, dc.jweTokenService)
	deleteAccountHandler := user.NewDeleteAccountHandler(dc.Repo, dc.slogLogger, dc.FileStorage, dc.VectorDB)
	updateUsernameHandler := user.NewUpdateUsernameHandler(dc.Repo, dc.Validator)
	getCurrentUserHandler := user.NewGetCurrentUserHandler(dc.Repo)
	updatePasswordHandler := user.NewChangePasswordHandler(dc.Repo, dc.Validator)
	forgotPasswordHandler := user.NewForgotPasswordHandler(dc.Repo, dc.mailerSender, dc.jweTokenService, dc.slogLogger, dc.Validator)
	resetPasswordHandler := user.NewResetPasswordHandler(dc.Repo, dc.jweTokenService, dc.slogLogger, dc.Validator)

	getChatsHandler := chat.NewGetChatsHandler(dc.Repo)
	getChatByIDHandler := chat.NewGetChatByIDHandler(dc.Repo)
	deleteChatHandler := chat.NewDeleteChatHandler(dc.Repo)

	uploadFileHandler := file.NewUploadFileHandler(dc.Repo, dc.FileStorage, dc.VectorDB, dc.Embedding, dc.Chunker, 48, true)
	getFilesHandler := file.NewGetFilesHandler(dc.Repo)
	deleteFileByIdHandler := file.NewDeleteFileHandler(dc.Repo, dc.FileStorage, dc.VectorDB)
	getFileHandler := file.NewGetFileHandler(dc.Repo, dc.FileStorage)

	sendMessageHandler := message.NewSendMessageHandler(message.Config{
		PastMessagesLimit: 5,
		Stream:            false,
		Repo:              dc.Repo,
		Ems:               dc.Embedding,
		Vb:                dc.VectorDB,
		ChatSvc:           dc.AIService,
	})

	generateQuizHandler := quiz.NewGenerateQuizHandler(dc.AIService, dc.Repo, dc.VectorDB)
	getQuizzesHandler := quiz.NewGetQuizzesHandler(dc.Repo)
	getQuizByIdHandler := quiz.NewGetQuizByIdHandler(dc.Repo)
	solveQuizHandler := quiz.NewSolveQuizHandler(dc.Repo)
	getQuizAttemptsHandler := quiz.NewGetQuizAttemptsHandler(dc.Repo)
	getQuizAttemptByIdHandler := quiz.NewGetAttemptByIdHandler(dc.Repo)

	getAllUsersHandler := admin.NewGetUsersHandler(dc.Repo)

	sl := dc.slogLogger
	app.Get("/healthcheck", Handle(healthcheckHandler, sl))
	app.Use(contextMiddleware)

	authApp := app.Group("/auth")
	authApp.Post("/signup", Handle(signupHandler, sl))
	authApp.Post("/login", Handle(loginHandler, sl))
	authApp.Post("/logout", Handle(logoutHandler, sl))
	authApp.Post("/refresh", Handle(refreshTokenHandler, sl))
	authApp.Post("/exchange", Handle(exchangeHandler, sl))
	authApp.Get("/google", googleOAuthService.RedirectHandler)
	authApp.Get("/facebook", facebookOAuthService.RedirectHandler)

	app.Get("/api/auth/google/callback", googleOAuthService.Callback)
	app.Get("/api/auth/facebook/callback", facebookOAuthService.Callback)

	usersPublicApp := app.Group("/users")
	usersPublicApp.Post("/forgot-password", Handle(forgotPasswordHandler, sl))
	usersPublicApp.Post("/reset-password", Handle(resetPasswordHandler, sl))

	usersApp := app.Group("/users", middlewareManager.AuthMiddleware)
	usersApp.Get("/profile", Handle(getCurrentUserHandler, sl))
	usersApp.Delete("/account", Handle(deleteAccountHandler, sl))
	usersApp.Patch("/account", Handle(updateUsernameHandler, sl))
	usersApp.Patch("/password", Handle(updatePasswordHandler, sl))

	chatsApp := app.Group("/chats", middlewareManager.AuthMiddleware)
	chatsApp.Get("/", Handle(getChatsHandler, sl))
	chatsApp.Get("/:id", Handle(getChatByIDHandler, sl))
	chatsApp.Delete("/:id", Handle(deleteChatHandler, sl))

	filesApp := app.Group("/files", middlewareManager.AuthMiddleware)
	filesApp.Post("/upload", Handle(uploadFileHandler, sl))
	filesApp.Get("/", Handle(getFilesHandler, sl))
	filesApp.Delete("/:id", Handle(deleteFileByIdHandler, sl))
	filesApp.Get("/:id", Handle(getFileHandler, sl))

	messagesApp := app.Group("/messages", middlewareManager.AuthMiddleware)
	messagesApp.Post("/", Handle(sendMessageHandler, sl))

	quizApp := app.Group("/quizzes", middlewareManager.AuthMiddleware)
	quizApp.Post("/", Handle(generateQuizHandler, sl))
	quizApp.Get("/", Handle(getQuizzesHandler, sl))
	quizApp.Get("/attempts", Handle(getQuizAttemptsHandler, sl))
	quizApp.Get("/attempts/:id", Handle(getQuizAttemptByIdHandler, sl))
	quizApp.Post("/:id/solve", Handle(solveQuizHandler, sl))
	quizApp.Get("/:id", Handle(getQuizByIdHandler, sl))

	adminApp := app.Group("/ebuhanife")
	adminApp.Get("/users", Handle(getAllUsersHandler, sl))

	if !domain.IsProdEnv() {
		app.Get("/swagger/*", fiberSwagger.WrapHandler)
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Welcome to MedAIist-Backend API",
			"code":    http.StatusOK,
		})
	})

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(domain.Error{
			Message: "endpoint not found",
			Code:    http.StatusNotFound,
		})
	})
}
