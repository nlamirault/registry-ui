package auth

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/cesanta/docker_auth/auth_server/authn"
	"github.com/cesanta/docker_auth/auth_server/server"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

var (
	ErrAlreadyExists = errors.New("account already exists")
	ConfigFile       string
	config           server.Config
)

func AddUser(account string, password string) error {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), 5)

	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	p := authn.PasswordString(string(pass))
	r := authn.Requirements{}
	r.Password = &p

	if _, ok := config.Users[account]; ok {
		return ErrAlreadyExists
	}

	config.Users[account] = &r

	b, err := yaml.Marshal(config)
	if err != nil {
		log.Fatalf("marshalling auth config: %v", err)
	}

	ioutil.WriteFile("./contrib/config/auth_config.yml", b, 0755)
	return nil
}

func ReadConfig(configFile string) error {
	ConfigFile = configFile

	b, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return fmt.Errorf("reading yml file: %v", err)
	}
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return fmt.Errorf("unmarshalling config: %v", err)
	}
	return nil
}
