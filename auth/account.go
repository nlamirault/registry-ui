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
	ErrNotExists     = errors.New("account not exists")
	ConfigFile       string
	Config           server.Config
)

func AddUser(account string, password string) error {
	pass, err := bcrypt.GenerateFromPassword([]byte(password), 5)

	if err != nil {
		log.Fatalf("hashing password: %v", err)
	}

	p := authn.PasswordString(string(pass))
	r := authn.Requirements{}
	r.Password = &p

	if _, ok := Config.Users[account]; ok {
		return ErrAlreadyExists
	}

	Config.Users[account] = &r

	b, err := yaml.Marshal(Config)
	if err != nil {
		log.Fatalf("marshalling auth config: %v", err)
	}

	ioutil.WriteFile(ConfigFile, b, 0755)
	return nil
}

func DeleteUser(account string) error {
	if _, ok := Config.Users[account]; !ok {
		return ErrNotExists
	}

	if account == "" {
		return fmt.Errorf("cannot delete anonymous account")
	}

	delete(Config.Users, account)

	b, err := yaml.Marshal(Config)
	if err != nil {
		log.Fatalf("marshalling auth config: %v", err)
	}

	ioutil.WriteFile(ConfigFile, b, 0755)
	return nil
}

func ReadConfig(configFile string) error {
	ConfigFile = configFile

	b, err := ioutil.ReadFile(ConfigFile)
	if err != nil {
		return fmt.Errorf("reading yml file: %v", err)
	}
	err = yaml.Unmarshal(b, &Config)
	if err != nil {
		return fmt.Errorf("unmarshalling config: %v", err)
	}
	return nil
}
