package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	adminV1 "go-wind-admin/api/gen/go/admin/service/v1"
	storageV1 "go-wind-admin/api/gen/go/storage/service/v1"

	"go-wind-admin/pkg/oss"
)

const (
	StateOK = "SUCCESS"
)

type UEditorService struct {
	adminV1.UEditorServiceHTTPServer

	log *log.Helper

	mc *oss.MinIOClient
}

func NewUEditorService(ctx *bootstrap.Context, mc *oss.MinIOClient) *UEditorService {
	return &UEditorService{
		log: ctx.NewLoggerHelper("ueditor/service/admin-service"),
		mc:  mc,
	}
}

func (s *UEditorService) UEditorAPI(ctx context.Context, req *storageV1.UEditorRequest) (*storageV1.UEditorResponse, error) {
	//s.log.Infof("UEditorAPI [%s]", req.GetAction())

	switch req.GetAction() {
	default:
		fallthrough
	case storageV1.UEditorAction_config.String():
		return &storageV1.UEditorResponse{
			//// 上传图片配置项
			//ImageActionName:     trans.Ptr("uploadImage"),                          // 执行上传图片的action名称
			//ImageFieldName:      trans.Ptr("file"),                                 // 提交的图片表单名称
			//ImageMaxSize:        trans.Ptr(int64(2048000)),                         // 上传大小限制，单位B
			//ImageAllowFiles:     []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}, // 上传图片格式显示
			//ImageCompressEnable: trans.Ptr(true),                                   // 是否压缩图片,默认是true
			//ImageCompressBorder: trans.Ptr(int64(1600)),                            // 图片压缩最长边限制
			//ImageInsertAlign:    trans.Ptr("none"),                                 // 插入的图片浮动方式
			//ImageUrlPrefix:      trans.Ptr(""),                                     /// 图片访问路径前缀
			//ImagePathFormat:     trans.Ptr(""),                                     // 上传保存路径,可以自定义保存路径和文件名格式
			//// 涂鸦图片上传配置项
			//ScrawlActionName:  trans.Ptr("uploadScrawl"), // 执行上传涂鸦的action名称
			//ScrawlFieldName:   trans.Ptr("file"),         // 提交的图片表单名称
			//ScrawlPathFormat:  trans.Ptr(""),             //上传保存路径,可以自定义保存路径和文件名格式
			//ScrawlMaxSize:     trans.Ptr(int64(2048000)), // 上传大小限制，单位B
			//ScrawlUrlPrefix:   trans.Ptr(""),             // 图片访问路径前缀
			//ScrawlInsertAlign: trans.Ptr("none"),         //
			//// 截图工具上传
			//SnapscreenActionName:  trans.Ptr("uploadImage"), // 执行上传截图的action名称
			//SnapscreenPathFormat:  trans.Ptr(""),            // 上传保存路径,可以自定义保存路径和文件名格式
			//SnapscreenUrlPrefix:   trans.Ptr(""),            // 图片访问路径前缀
			//SnapscreenInsertAlign: trans.Ptr("none"),        // 插入的图片浮动方式
			//// 抓取远程图片配置
			//CatcherLocalDomain: []string{"127.0.0.1", "localhost", "img.baidu.com"},
			//CatcherActionName:  trans.Ptr("catchImage"),                           // 执行抓取远程图片的action名称
			//CatcherFieldName:   trans.Ptr("file"),                                 // 提交的图片列表表单名称
			//CatcherPathFormat:  trans.Ptr(""),                                     // 上传保存路径,可以自定义保存路径和文件名格式
			//CatcherUrlPrefix:   trans.Ptr(""),                                     // 图片访问路径前缀
			//CatcherMaxSize:     trans.Ptr(int64(2048000)),                         // 上传大小限制，单位B
			//CatcherAllowFiles:  []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}, // 抓取图片格式显示
			//// 上传视频配置
			//VideoActionName: trans.Ptr("uploadVideo"),    // 执行上传视频的action名称
			//VideoFieldName:  trans.Ptr("file"),           // 提交的视频表单名称
			//VideoPathFormat: trans.Ptr(""),               // 上传保存路径,可以自定义保存路径和文件名格式
			//VideoUrlPrefix:  trans.Ptr(""),               // 视频访问路径前缀
			//VideoMaxSize:    trans.Ptr(int64(102400000)), // 上传大小限制，单位B，默认100MB
			//VideoAllowFiles: []string{".flv", ".swf", ".mkv", ".avi", ".rm", ".rmvb", ".mpeg", ".mpg",
			//	".ogg", ".ogv", ".mov", ".wmv", ".mp4", ".webm", ".mp3", ".wav", ".mid"}, // 上传视频格式显示
			//// 上传文件配置
			//FileActionName: trans.Ptr("uploadFile"),
			//FileFieldName:  trans.Ptr("file"),
			//FilePathFormat: trans.Ptr(""),
			//FileUrlPrefix:  trans.Ptr(""),
			//FileMaxSize:    trans.Ptr(int64(51200000)),
			//FileAllowFiles: []string{".png", ".jpg", ".jpeg", ".gif", ".bmp",
			//	".flv", ".swf", ".mkv", ".avi", ".rm", ".rmvb", ".mpeg", ".mpg",
			//	".ogg", ".ogv", ".mov", ".wmv", ".mp4", ".webm", ".mp3", ".wav", ".mid",
			//	".rar", ".zip", ".tar", ".gz", ".7z", ".bz2", ".cab", ".iso",
			//	".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt", ".md", ".xml"},
			//// 列出指定目录下的图片
			//ImageManagerActionName:  trans.Ptr("listImage"),
			//ImageManagerListPath:    trans.Ptr(""),
			//ImageManagerListSize:    trans.Ptr(int64(20)),
			//ImageManagerUrlPrefix:   trans.Ptr(""),
			//ImageManagerInsertAlign: trans.Ptr("none"),
			//ImageManagerAllowFiles:  []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"},
			//// 列出指定目录下的文件
			//FileManagerActionName: trans.Ptr("listFile"),
			//FileManagerListPath:   trans.Ptr(""),
			//FileManagerUrlPrefix:  trans.Ptr(""),
			//FileManagerListSize:   trans.Ptr(int64(20)),
			//FileManagerAllowFiles: []string{".png", ".jpg", ".jpeg", ".gif", ".bmp",
			//	".flv", ".swf", ".mkv", ".avi", ".rm", ".rmvb", ".mpeg", ".mpg",
			//	".ogg", ".ogv", ".mov", ".wmv", ".mp4", ".webm", ".mp3", ".wav", ".mid",
			//	".rar", ".zip", ".tar", ".gz", ".7z", ".bz2", ".cab", ".iso",
			//	".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".pdf", ".txt", ".md", ".xml"},
		}, nil

	case storageV1.UEditorAction_listFile.String():
		return s.mc.ListFileForUEditor(ctx, "files", "")

	case storageV1.UEditorAction_listImage.String():
		return s.mc.ListFileForUEditor(ctx, "images", "")
	}
}
