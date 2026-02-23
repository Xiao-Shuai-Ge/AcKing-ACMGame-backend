package logic

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"tgwp/global"
	"tgwp/log/zlog"
	"tgwp/model"
	"tgwp/repo"
	"tgwp/response"
	"tgwp/types"
	"tgwp/utils/email"
	"tgwp/utils/jwtUtils"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginLogic struct {
}

const (
	EMAIL_REGEX      = "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
	REDIS_EMAIL_CODE = "login:email:%s:code"
)

func NewLoginLogic() *LoginLogic {
	return &LoginLogic{}
}

func (l *LoginLogic) SendCode(ctx context.Context, req types.SendCodeReq) (resp types.SendCodeResp, err error) {
	if req.Email == "" {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	re := regexp.MustCompile(EMAIL_REGEX)
	if isMatch := re.MatchString(req.Email); !isMatch {
		return resp, response.ErrResp(err, response.EMAIL_NOT_VALID)
	}
	if global.Rdb == nil {
		return resp, response.ErrResp(errors.New("redis not init"), response.REDIS_ERROR)
	}
	code := rand.Intn(1000000)
	err = global.Rdb.Set(ctx, fmt.Sprintf(REDIS_EMAIL_CODE, req.Email), code, 5*time.Minute).Err()
	if err != nil {
		return resp, response.ErrResp(err, response.REDIS_ERROR)
	}
	err = email.SendCode(req.Email, int64(code))
	if err != nil {
		return resp, response.ErrResp(err, response.EMAIL_SEND_ERROR)
	}
	return resp, nil
}

func (l *LoginLogic) Register(ctx context.Context, req types.RegisterReq) (resp types.LoginResp, err error) {
	if req.Email == "" || req.Password == "" || req.Username == "" || req.Code == "" {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	re := regexp.MustCompile(EMAIL_REGEX)
	if isMatch := re.MatchString(req.Email); !isMatch {
		return resp, response.ErrResp(err, response.EMAIL_NOT_VALID)
	}
	if global.Rdb == nil {
		return resp, response.ErrResp(errors.New("redis not init"), response.REDIS_ERROR)
	}
	code, err := global.Rdb.Get(ctx, fmt.Sprintf(REDIS_EMAIL_CODE, req.Email)).Int()
	if err != nil {
		return resp, response.ErrResp(err, response.VERIFY_CODE_VALID)
	}
	if fmt.Sprintf("%06d", code) != req.Code {
		return resp, response.ErrResp(err, response.VERIFY_CODE_VALID)
	}
	userRepo := repo.NewUserRepo(global.DB)
	exist, err := userRepo.GetByEmail(req.Email)
	if err == nil && exist.ID != 0 {
		return resp, response.ErrResp(errors.New("user exists"), response.USER_ALREADY_EXISTS)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zlog.CtxErrorf(ctx, "GetByEmail err: %v", err)
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		zlog.CtxErrorf(ctx, "bcrypt err: %v", err)
		return resp, response.ErrResp(err, response.INTERNAL_ERROR)
	}
	user := model.User{
		Email:     req.Email,
		Password:  string(hashed),
		Username:  req.Username,
		AvatarUrl: req.AvatarUrl,
		Rating:    req.Rating,
	}
	if err = userRepo.Create(&user); err != nil {
		zlog.CtxErrorf(ctx, "Create user err: %v", err)
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	token, err := jwtUtils.GenToken(user.ID, user.Username, global.ROLE_USER, global.ATOKEN_EFFECTIVE_TIME)
	if err != nil {
		zlog.CtxErrorf(ctx, "GenToken err: %v", err)
		return resp, response.ErrResp(err, response.INTERNAL_ERROR)
	}
	return types.LoginResp{
		Token: token,
		User: toUserInfo(user),
	}, nil
}

func (l *LoginLogic) Login(ctx context.Context, req types.LoginReq) (resp types.LoginResp, err error) {
	if req.Email == "" || req.Password == "" {
		return resp, response.ErrResp(errors.New("param blank"), response.PARAM_NOT_COMPLETE)
	}
	userRepo := repo.NewUserRepo(global.DB)
	user, err := userRepo.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, response.ErrResp(err, response.MEMBER_NOT_EXIST)
		}
		zlog.CtxErrorf(ctx, "GetByEmail err: %v", err)
		return resp, response.ErrResp(err, response.DATABASE_ERROR)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return resp, response.ErrResp(err, response.PASSWORD_ERROR)
	}
	token, err := jwtUtils.GenToken(user.ID, user.Username, global.ROLE_USER, global.ATOKEN_EFFECTIVE_TIME)
	if err != nil {
		zlog.CtxErrorf(ctx, "GenToken err: %v", err)
		return resp, response.ErrResp(err, response.INTERNAL_ERROR)
	}
	return types.LoginResp{
		Token: token,
		User:  toUserInfo(user),
	}, nil
}

func toUserInfo(user model.User) types.UserInfo {
	return types.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		AvatarUrl: user.AvatarUrl,
		Rating:    user.Rating,
	}
}
