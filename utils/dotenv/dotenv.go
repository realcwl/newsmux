package dotenv

import (
	"os"

	"github.com/joho/godotenv"
)

// Load loads the .env file following the convention: https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
// It only need to be called once in main function, other code can use env through os.Getenv('ENV_NAME') during runtime
func LoadDotEnvs() error {
	// check whether running in development, testing, production etc.
	env := os.Getenv("NEWSMUX_ENV")
	if env == "" {
		env = "dev"
	}

	// .env.[runtime_env].local has highest priority, usually contains username and password and other sensitive information
	godotenv.Load(".env." + env + ".local")
	godotenv.Load(".env." + ".local")
	// .env.[runtime_env] usually contains db connection information
	godotenv.Load(".env." + env)
	// .env usually contains shared variables(which might be overwritten by envs above)
	godotenv.Load(".env")
	return nil
}
