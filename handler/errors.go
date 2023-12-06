package handler

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type CustomError struct {
	Errno    int
	ErrorMsg string
}

// 实现 error 接口的 Error 方法
func (e CustomError) Error() string {
	return fmt.Sprintf("错误码: %d, 描述: %s", e.Errno, e.ErrorMsg)
}

// 定义错误常量
var (
	ErrSuccess                = CustomError{Errno: 0, ErrorMsg: "请求成功"}
	ErrInvalidParameter       = CustomError{Errno: 2, ErrorMsg: "参数错误"}
	ErrAccessTokenExpired     = CustomError{Errno: 111, ErrorMsg: "access token 失效"}
	ErrAuthenticationFailed   = CustomError{Errno: -6, ErrorMsg: "身份验证失败"}
	ErrUnauthorizedUserAccess = CustomError{Errno: 6, ErrorMsg: "不允许接入用户数据"}
	ErrApiRateLimitExceeded   = CustomError{Errno: 31034, ErrorMsg: "命中接口频控"}
	ErrShareNotFound          = CustomError{Errno: 2131, ErrorMsg: "该分享不存在"}
	ErrDuplicateFile          = CustomError{Errno: 10, ErrorMsg: "转存文件已经存在"}
	ErrFileNotFound           = CustomError{Errno: -3, ErrorMsg: "文件不存在"}
	ErrFileNotExist           = CustomError{Errno: -31066, ErrorMsg: "文件不存在"}
	ErrSelfSentShare          = CustomError{Errno: 11, ErrorMsg: "自己发送的分享"}
	ErrExcessiveTransferCount = CustomError{Errno: 255, ErrorMsg: "转存数量太多"}
	ErrBatchTransferError     = CustomError{Errno: 12, ErrorMsg: "批量转存出错"}
	ErrExpiredRights          = CustomError{Errno: -1, ErrorMsg: "权益已过期"}
)

func HandlerGlobalErrors() {
	if r := recover(); r != nil {
		logrus.Error("Error Occured: ", r)
	}
}
