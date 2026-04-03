package docs

import swagv1 "github.com/swaggo/swag"

func init() {
	SwaggerInfo.Schemes = []string{"http", "https"}
	swagv1.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
