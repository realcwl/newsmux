package dotenv

import (
	"os"
	"regexp"

	"github.com/joho/godotenv"
)

// Load loads the .env file following the convention: https://github.com/bkeepers/dotenv#what-other-env-files-can-i-use
// It only need to be called once in main function, other code can use env through os.Getenv('ENV_NAME') during runtime
func LoadDotEnvs() error {
	// check whether running in development, testing, production etc.
	loadDotEnvs("")
	return nil
}

func loadDotEnvs(rootPath string) {
	env := os.Getenv("NEWSMUX_ENV")
	if env == "" {
		env = "dev"
	}

	// .env.[runtime_env].local has highest priority, usually contains username and password and other sensitive information
	godotenv.Load(rootPath + ".env." + env + ".local")
	godotenv.Load(rootPath + ".env.local")
	// .env.[runtime_env] usually contains db connection information
	godotenv.Load(rootPath + ".env." + env)
	// .env usually contains shared variables(which might be overwritten by envs above)
	godotenv.Load(rootPath + ".env")
}

// Have to write this helper function due to a known issue of godotenv
// https://github.com/joho/godotenv/issues/43
func LoadDotEnvsInTests() error {
	re := regexp.MustCompile(`^(.*newsmux)`)
	cwd, _ := os.Getwd()
	rootPath := re.Find([]byte(cwd))

	godotenv.Load(string(rootPath) + "/" + ".env.test")
	return nil
}
