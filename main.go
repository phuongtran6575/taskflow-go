package main

import (
	"TaskFlow-Go/internal/app"

	_ "TaskFlow-Go/docs"
)

//	@title			TaskFlow API
//	@version		1.0
//	@description	Task management API with workspace, project, and team collaboration features.
//	@termsOfService	https://taskflow.io/terms

//	@contact.name	TaskFlow Support
//	@contact.email	support@taskflow.io

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer " followed by your JWT token.

func main() {
	app.Run()
}
