package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/TwinProduction/go-color"
	"github.com/atotto/clipboard"
	"gopkg.in/ini.v1"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

const version string = "1.1.0"

func main() {
	env := flag.String("e", "dev", "Config environment")
	writeMode := flag.Bool("w", false, "Login with write permissions user")
	bversion := flag.Bool("v", false, "Version info")
	flag.Parse()

	rdsAuthIniKeys := []string{"db", "region", "user"}
	awsCredentialsKeys := []string{"role_arn", "role_session_name", "source_profile"}

	database, awsRegion, dbUser := readConfig(*env, ".rdsauth.ini", rdsAuthIniKeys)
	roleARN, roleSessionName, sourceProfile := readConfig(*env, ".aws/credentials", awsCredentialsKeys)

	var configProfile string

	if (*bversion) {
		fmt.Printf(color.Blue + "rdsauth version: %s\n" + color.Reset, version)
		os.Exit(0)
	}

	if roleARN != "" {
		configProfile = "rdsauth"
		createTempProfiles(dbUser, roleARN, roleSessionName, sourceProfile)
	}else{
		configProfile = *env
		fmt.Printf(color.Cyan + "Profile: %s\n" + color.Reset, *env)
	}

	homeDir := os.Getenv("HOME")
	credentialsFile := fmt.Sprintf("%s/.aws/credentials", homeDir)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
					     config.WithRegion(awsRegion),
					     config.WithSharedCredentialsFiles([]string{"/tmp/rdsauthcredentials",credentialsFile}),
					     config.WithSharedConfigProfile(configProfile))
	if err != nil {
		log.Fatalf(
			color.Red +
			"Error: failed to load configuration: %s\n" +
			color.Yellow +
			"Have you created an ~/.aws/credentials file\n" +
			color.Reset,
			err,
		)
	}


	fmt.Printf(color.Cyan + "Running in region: %s\n" + color.Reset, cfg.Region)
	if (cfg.Region == ""){
		log.Fatalf(color.Red + "No environment named: '%s' found! Check the name and your ~/.rdsauth.ini file" + color.Reset, *env)
	}

	getAuth(cfg, database, awsRegion, dbUser, *writeMode)
}


func createTempProfiles(user, roleARN, roleSessionName, sourceProfile string) {
	username := strings.Split(user, "")[1]
	roleARNBase := strings.Split(roleARN, "role/")[0]
	rdsRoleARN := fmt.Sprintf("%srole/RDS_%s@<ROLENAME>", roleARNBase, username)

	fmt.Printf(color.Cyan + "%s\n" + color.Reset, rdsRoleARN)

	tempCredContent := fmt.Sprintf("[rdsauth]\nrole_arn = %s\nrole_session_name = %s\nsource_profile = %s\nalias=rdsauth", rdsRoleARN, roleSessionName, sourceProfile)
	err := os.WriteFile("/tmp/rdsauthcredentials", []byte(tempCredContent) , 0644)
	if err != nil {
		log.Fatalf(color.Red + "Error: failed to create temp credentials file" + color.Reset)
	}
}


func readConfig(env, filename string, fieldList []string) (string, string, string){
	homeDir := os.Getenv("HOME")
	configLocation := fmt.Sprintf("%s/%s", homeDir, filename)
	cfg, err := ini.Load(configLocation)
	if err != nil {
		log.Fatalf(
			color.Red +
			"Failed to read file: %v" +
			color.Reset,
			err,
		)
	}

	a := cfg.Section(env).Key(fieldList[0]).String()
	b := cfg.Section(env).Key(fieldList[1]).String()
	c := cfg.Section(env).Key(fieldList[2]).String()

	return a, b, c
}


func getAuth(cfg aws.Config, database, awsRegion, dbUser string, writeMode bool) {
	if (writeMode == true) {
		writeUsername := fmt.Sprintf("%s_write", dbUser)
		dbUser = writeUsername
	}
	fmt.Printf(color.Cyan + "Fetching token for user: %s\n" + color.Reset, dbUser)

	if (dbUser == ""){
		log.Fatalf(color.Red + "No username found. Check your ~/.rdsauth.ini file" + color.Reset)
	}

	authenticationToken, err := auth.BuildAuthToken(context.TODO(),
	database,
	awsRegion,
	dbUser,
	cfg.Credentials)

	if err != nil {
		log.Fatalf(
			color.Red +
			"Error: failed to create authentication token: %s\n" +
			color.Yellow +
			"Have you created an ~/.aws/credentials file\n" +
			color.Reset,
			err,
		)
	}

	fmt.Println(authenticationToken)
	clipboard.WriteAll(authenticationToken)
	fmt.Println(color.Green + "Token copied to clipboard!" + color.Reset)
}
