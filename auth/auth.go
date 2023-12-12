package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/config"
	"github.com/wangxso/backuptool/db"
	openapiclient "github.com/wangxso/backuptool/openxpanapi"
)

type authReturnType struct {
	ExpiresIn     int    `json:"expires_in"`
	RefreshToken  string `json:"refresh_token"`
	AccessToken   string `json:"access_token"`
	SessionSecret string `json:"session_secret"`
	SessionKey    string `json:"session_key"`
	Scope         string `json:"scope"`
}

const (
	AuthCodeValidity   = 10 * time.Minute    // 授权码有效期
	AccessCodeValidity = 30 * 24 * time.Hour // Access Code 有效期
)

func getAcessToken(authCode string, clientId string, clientSecret string, redirectUri string) string {

	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	resp, r, err := api_client.AuthApi.OauthTokenCode2token(context.Background()).Code(authCode).ClientId(clientId).ClientSecret(clientSecret).RedirectUri(redirectUri).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthApi.OauthTokenCode2token``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OauthTokenCode2token`: OauthTokenAuthorizationCodeResponse
	fmt.Fprintf(os.Stdout, "Response from `AuthApi.OauthTokenCode2token`: %v\n", resp)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", r)
	}
	return string(bodyBytes)
}

func Login(authCode string) authReturnType {
	ctx := db.Client.Context()
	appKey := config.BackUpConfig.BaiduDisk.AppKey
	appSecret := config.BackUpConfig.BaiduDisk.SecretKey
	redirectUri := config.BackUpConfig.BaiduDisk.RedirectUri

	respStr := getAcessToken(authCode, appKey, appSecret, redirectUri)

	var resp authReturnType
	err := json.Unmarshal([]byte(respStr), &resp)
	if err != nil {
		log.Fatal(err)
	}
	logrus.Info("Login Success AccessCode: ", resp.AccessToken)
	db.Client.Set(ctx, "AccessCode", resp.AccessToken, AccessCodeValidity)
	db.Client.Set(ctx, "RefreshCode", resp.RefreshToken, AccessCodeValidity*2)
	return resp
}
