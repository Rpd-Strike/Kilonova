package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Login

func (s *BaseAPI) Login(ctx context.Context, uname, pwd string) (int, *StatusError) {
	user, err := s.db.UserByName(ctx, uname)
	if err != nil {
		zap.S().Warn(err)
		return -1, Statusf(400, "Invalid login details")
	}
	// Maybe the user is trying to log in by email
	if user == nil {
		user, err = s.db.UserByEmail(ctx, uname)
		if err != nil {
			zap.S().Warn(err)
			return -1, Statusf(400, "Invalid login details")
		}
	}

	if user == nil {
		return -1, Statusf(400, "Invalid login details")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pwd))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return -1, Statusf(400, "Invalid login details")
	} else if err != nil {
		// This should never happen. It means that bcrypt suffered something
		zap.S().Warn(err)
		return -1, ErrUnknownError
	}

	return user.ID, nil
}

// Signup

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (s *BaseAPI) Signup(ctx context.Context, email, uname, pwd, lang string) (int, *StatusError) {
	uname = strings.TrimSpace(uname)
	if !(len(uname) >= 3 && len(uname) <= 32 && usernameRegex.MatchString(uname)) {
		return -1, Statusf(400, "Username must be between 3 and 32 characters long and must contain only letters, digits, underlines and dashes.")
	}
	if err := s.CheckValidPassword(pwd); err != nil {
		return -1, err
	}
	if !(lang == "" || lang == "en" || lang == "ro") {
		return -1, Statusf(400, "Invalid language.")
	}
	if !govalidator.IsExistingEmail(email) {
		return -1, Statusf(400, "Invalid email.")
	}

	if exists, err := s.db.UserExists(ctx, uname, email); err != nil || exists {
		return -1, Statusf(400, "User matching email or username already exists!")
	}

	if lang == "" {
		lang = config.Common.DefaultLang
	}

	id, err := s.createUser(ctx, uname, email, pwd, lang)
	if err != nil {
		fmt.Println(err)
		return -1, Statusf(500, "Couldn't create user")
	}

	user, err1 := s.UserFull(ctx, id)
	if err1 != nil {
		fmt.Println(err1)
		return -1, err1
	}

	if err := s.SendVerificationEmail(ctx, user.ID, user.Name, user.Email); err != nil {
		zap.S().Info("Couldn't send user verification email:", err)
	}

	return id, nil
}

func (s *BaseAPI) CheckValidPassword(pwd string) *StatusError {
	if len(pwd) < 6 || len(pwd) > 128 {
		return Statusf(400, "Invalid password length.")
	}
	return nil
}