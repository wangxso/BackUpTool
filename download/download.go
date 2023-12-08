package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cheggaaa/pb/v3"
	"github.com/sirupsen/logrus"
	"github.com/wangxso/backuptool/db"
	openapiclient "github.com/wangxso/backuptool/openxpanapi"
)

type FileReturn struct {
	TkbindId       int    `json:"tkbind_id"`
	OwnerType      int    `json:"owner_type"`
	RealCategory   string `json:"real_category"`
	ServerFileName string `json:"server_filename"`
	Privacy        int    `json:"privacy"`
	Category       int    `json:"category"`
	Unlist         int    `json:"unlist"`
	FsId           int64  `json:"fs_id"`
	DirEmpty       int    `json:"dir_empty"`
	ServerAtime    int64  `json:"server_atime"`
	ServerCtime    int64  `json:"server_ctime"`
	LocalMtime     int64  `json:"local_mtime"`
	Size           int64  `json:"size"`
	Isdir          int    `json:"isdir"`
	Share          int    `json:"share"`
	Path           string `json:"path"`
	LocalCtime     int64  `json:"local_ctime"`
	ServerMtime    int64  `json:"server_mtime"`
	Empty          int    `json:"empty"`
	OperId         int64  `json:"oper_id"`
	MD5            string `json:"md5"`
}

type FileListReturn struct {
	ErrorNo   int          `json:"errno"`
	GuidInfo  string       `json:"guid_info"`
	List      []FileReturn `json:"list"`
	RequestId int64        `json:"request_id"`
	Guid      int          `json:"guid"`
}

type FileItem struct {
	Category       int               `json:"category"`
	FsID           int64             `json:"fs_id"`
	IsDir          int               `json:"isdir"`
	LocalCtime     int64             `json:"local_ctime"`
	LocalMtime     int64             `json:"local_mtime"`
	MD5            string            `json:"md5"`
	Path           string            `json:"path"`
	ServerCtime    int64             `json:"server_ctime"`
	ServerFilename string            `json:"server_filename"`
	ServerMtime    int64             `json:"server_mtime"`
	Size           int64             `json:"size"`
	Thumbs         map[string]string `json:"thumbs"`
}

type FileMultiListReturn struct {
	Cursor    int        `json:"cursor"`
	Errmsg    string     `json:"errmsg"`
	Errno     int        `json:"errno"`
	HasMore   int        `json:"has_more"`
	List      []FileItem `json:"list"`
	RequestID string     `json:"request_id"`
}

// ProgressWriter 实现了io.Writer接口，用于显示下载进度
type ProgressWriter struct {
	Total     int64 // 要下载的文件的总大小
	Completed int64 // 已下载的文件大小
}

// Write 实现了io.Writer接口的Write方法
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.Completed += int64(n)
	pw.DisplayProgress()
	return n, nil
}

// DisplayProgress 显示下载进度
func (pw *ProgressWriter) DisplayProgress() {
	progress := float64(pw.Completed) / float64(pw.Total) * 100
	logrus.Infof("下载进度: %.2f%%\r", progress)
}

// GetFileList
// dir: /来自：back设备
// limit: int; desc int; order string(time); start string("0");forlder string("0");
func GetFileList(accessToken, dir, order, start, folder string, limit, desc int32) FileListReturn {
	web := "" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	api_client := openapiclient.NewAPIClient(configuration)
	_, r, err := api_client.FileinfoApi.Xpanfilelist(context.Background()).AccessToken(accessToken).Folder(folder).Web(web).Start(start).Limit(limit).Dir(dir).Order(order).Desc(desc).Execute()
	if err != nil {
		logrus.Error("Error when calling `FileinfoApi.Xpanfilelist``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// logrus.Info("Response from `FileinfoApi.Xpanfilelist`: ", resp)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}
	var response FileListReturn
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func GetMultiFileList(accessToken, path string, recursion int, order string, desc int, start int, limit int) FileMultiListReturn {

	configuration := openapiclient.NewConfiguration()
	configuration.Debug = true
	api_client := openapiclient.NewAPIClient(configuration)
	_, r, err := api_client.MultimediafileApi.Xpanfilelistall(context.Background()).AccessToken(accessToken).Path(path).Recursion(int32(recursion)).Start(int32(start)).Limit(int32(limit)).Order(order).Desc(int32(desc)).Execute()
	if err != nil {
		logrus.Error("Error when calling `MultimediafileApi.Xpanfilelistall``: ", err)
		logrus.Error("Full HTTP response: ", r)
	}
	// logrus.Info("Response from `MultimediafileApi.Xpanfilelistall`: ", resp)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.Error("err: ", r)
	}
	var response FileMultiListReturn
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatal(err)
	}
	return response
}

func GetDlink(accessToken string, fsids []uint64) ([]map[string]string, error) {

	// 如果是查询共享目录或专属空间内文件时需要path，可结合官网文档
	path := ""
	dlinks := make([]map[string]string, 0)

	// call Api
	arg := NewFileMetasArg(fsids, path)
	ret, err := FileMetas(accessToken, arg)
	if err != nil {
		logrus.Error("[msg: filemetas error] err:", err.Error())
		return nil, err
	} else {
		// fmt.Printf("ret:%+v", ret)
		logrus.Infof("ret.List:%+v", ret.List)
		// 获取list的第一个元素的dlink示例
		for _, v := range ret.List {
			item := make(map[string]string, 0)
			item["dlink"] = v.Dlink
			item["filename"] = v.Filename
			dlinks = append(dlinks, item)
		}
	}
	return dlinks, nil
}

func Download(fid uint64, targetPath string) error {
	redisCli := db.Client
	accessToken, _ := redisCli.Get(redisCli.Context(), "AccessCode").Result()

	dlink, err := GetDlink(accessToken, []uint64{fid})
	if err != nil {
		logrus.Error(err)
		return err
	}
	uri := fmt.Sprintf("%s&access_token=%s", dlink[0]["dlink"], accessToken)
	filename := dlink[0]["filename"]
	// 发起HTTP GET请求
	resp, err := http.Get(uri)
	if err != nil {
		logrus.Error("无法下载文件:", err)
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP响应状态码
	if resp.StatusCode != http.StatusOK {
		logrus.Error("下载请求失败:", resp.Status)
		return err
	}

	// 创建保存文件的本地文件
	filename = fmt.Sprintf("%s/%s", targetPath, filename)
	out, err := os.Create(filename) // 替换为您要保存的本地文件路径
	if err != nil {
		logrus.Error("无法创建文件:", err)
		return err
	}
	defer out.Close()
	fileSize := resp.ContentLength

	// 创建一个进度条
	progressBar := pb.Full.Start64(fileSize)
	progressBar.Set(pb.Bytes, true)

	// 创建一个多写器，用于同时将数据写入文件和进度条
	writer := io.MultiWriter(out, progressBar.NewProxyWriter(io.Discard))

	// 创建一个限速读取器，用于限制下载速度（可选）
	limitReader := &io.LimitedReader{
		R: resp.Body,
		N: fileSize,
	}

	// 将HTTP响应体复制到本地文件，并显示下载进度
	_, err = io.Copy(writer, limitReader)
	if err != nil {
		logrus.Error(err)
	}
	// 完成进度条
	progressBar.Finish()
	return nil
}
