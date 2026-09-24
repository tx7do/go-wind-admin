package oss

// 本文件补充 utils.go 中三个 MIME/扩展名映射函数未被 utils_test.go 覆盖到的分支：
//   - ContentTypeToBucketName：application 子类型的 office 相关子串匹配、
//     解析失败（含空白）回退的小写化、multipart 等主类型的默认落桶；
//   - FileExtensionToBucketName：各扩展名 switch 臂的完整覆盖与 MIME 透传；
//   - ContentTypeToFileExtension：各显式映射臂的完整覆盖、
//     解析失败（含空白）回退的小写化。
//
// 以上均为纯字符串映射，结果确定性断言。

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContentTypeToBucketName_OfficeSubstringsAndFallbacks 补充覆盖
// application 子类型的 office 子串匹配分支、大小写回退与解析失败回退。
func TestContentTypeToBucketName_OfficeSubstringsAndFallbacks(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		// 子串匹配 office 文档（word/excel/powerpoint/officedocument）
		{"word substring", "application/x-word-foo", BucketDocs},
		{"excel substring", "application/x-excel-foo", BucketDocs},
		{"powerpoint substring", "application/x-powerpoint-foo", BucketDocs},
		{"officedocument substring", "application/x-officedocument-foo", BucketDocs},
		{"vnd.ms- prefix", "application/vnd.ms-foo", BucketDocs},

		// 大写 MIME 经小写化后命中主类型
		{"uppercase video", "VIDEO/MP4", BucketVideos},
		{"uppercase audio", "AUDIO/WAV", BucketAudios},
		{"uppercase text", "TEXT/CSS", BucketDocs},

		// 带 MIME 参数：参数被剥离
		{"image with params", "image/png; name=\"foo.png\"", BucketImages},

		// 解析成功（标准库 TrimSpace 归一空白）后命中 image 主类型
		{"whitespace padded image", "  image/png  ", BucketImages},

		// 解析失败（无子类型）：畸形串不再推断主类型，直接落默认 files 桶
		{"trailing slash parse failure defaults to files", "image/", BucketFiles},

		// multipart 等其它主类型落入默认 files
		{"multipart main type", "multipart/form-data", BucketFiles},
		{"model main type", "model/gltf-binary", BucketFiles},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ContentTypeToBucketName(tt.contentType),
				"ContentTypeToBucketName(%q)", tt.contentType)
		})
	}
}

// TestFileExtensionToBucketName_AllArms 补充覆盖扩展名 switch 的全部臂
// （既有 utils_test.go 仅命中 jpg/png/mp4/mp3/js/json/pdf，此处补齐其余映射与默认落桶）。
func TestFileExtensionToBucketName_AllArms(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		want string
	}{
		// images
		{"bmp", "bmp", BucketImages},
		{"ico", "ico", BucketImages},
		{"svg", "svg", BucketImages},
		{"tif", "tif", BucketImages},
		{"tiff", "tiff", BucketImages},
		{"heic", "heic", BucketImages},
		{"gif", "gif", BucketImages},
		{"webp", "webp", BucketImages},
		{"jpeg", "jpeg", BucketImages},

		// videos
		{"mov", "mov", BucketVideos},
		{"mkv", "mkv", BucketVideos},
		{"avi", "avi", BucketVideos},
		{"flv", "flv", BucketVideos},
		{"mpeg", "mpeg", BucketVideos},
		{"mpg", "mpg", BucketVideos},
		{"webm", "webm", BucketVideos},

		// audios
		{"wav", "wav", BucketAudios},
		{"ogg", "ogg", BucketAudios},
		{"m4a", "m4a", BucketAudios},
		{"flac", "flac", BucketAudios},
		{"aac", "aac", BucketAudios},

		// text / docs
		{"txt", "txt", BucketDocs},
		{"html", "html", BucketDocs},
		{"htm", "htm", BucketDocs},
		{"css", "css", BucketDocs},
		{"csv", "csv", BucketDocs},
		{"md", "md", BucketDocs},
		{"xml", "xml", BucketDocs},
		{"doc", "doc", BucketDocs},
		{"docx", "docx", BucketDocs},
		{"xls", "xls", BucketDocs},
		{"xlsx", "xlsx", BucketDocs},
		{"ppt", "ppt", BucketDocs},
		{"pptx", "pptx", BucketDocs},

		// archives / binaries → files
		{"zip", "zip", BucketFiles},
		{"tar", "tar", BucketFiles},
		{"gz", "gz", BucketFiles},
		{"tgz", "tgz", BucketFiles},
		{"7z", "7z", BucketFiles},
		{"rar", "rar", BucketFiles},
		{"bz2", "bz2", BucketFiles},
		{"bin", "bin", BucketFiles},
		{"exe", "exe", BucketFiles},

		// MIME 透传（含斜杠 → 复用 ContentTypeToBucketName）
		{"mime passthrough video", "video/quicktime", BucketVideos},
		{"mime passthrough audio", "audio/ogg", BucketAudios},

		// 未匹配默认 files
		{"unknown ext", "someunknownext", BucketFiles},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, FileExtensionToBucketName(tt.ext),
				"FileExtensionToBucketName(%q)", tt.ext)
		})
	}
}

// TestContentTypeToFileExtension_ExplicitArms 补充覆盖显式映射 switch 的其余臂，
// 以及解析失败（空白前缀）回退的小写化命中。
func TestContentTypeToFileExtension_ExplicitArms(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		// images
		{"image/jpg", "image/jpg", "jpg"},
		{"image/gif", "image/gif", "gif"},
		{"image/webp", "image/webp", "webp"},
		{"image/bmp", "image/bmp", "bmp"},
		{"image/vnd.microsoft.icon", "image/vnd.microsoft.icon", "ico"},

		// videos
		{"video/webm", "video/webm", "webm"},
		{"video/quicktime", "video/quicktime", "mov"},
		{"video/x-matroska", "video/x-matroska", "mkv"},
		{"video/mkv", "video/mkv", "mkv"},

		// audios
		{"audio/wav", "audio/wav", "wav"},
		{"audio/x-wav", "audio/x-wav", "wav"},
		{"audio/ogg", "audio/ogg", "ogg"},
		{"audio/vorbis", "audio/vorbis", "ogg"},
		{"audio/mp4", "audio/mp4", "m4a"},

		// text
		{"text/csv", "text/csv", "csv"},

		// office documents
		{"application/msword", "application/msword", "doc"},
		{"wordprocessingml document", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx"},
		{"vnd.ms-excel", "application/vnd.ms-excel", "xls"},
		{"spreadsheetml sheet", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"},
		{"vnd.ms-powerpoint", "application/vnd.ms-powerpoint", "ppt"},
		{"presentationml presentation", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx"},

		// 解析成功（标准库 TrimSpace 归一空白）后命中映射
		{"whitespace padded image", "  image/png  ", "png"},

		// 解析失败（无子类型）：mt 回退保留 "image/"，无任何映射命中，返回空串
		{"trailing slash parse failure", "image/", ""},

		// 不在显式 switch、但 mime.ExtensionsByType 有映射的回退路径
		//（标准库返回恒带点，函数统一剥点后返回）
		{"image/tiff via mime fallback", "image/tiff", "tif"},

		// 无映射：返回空串
		{"unmapped application subtype", "application/x-custom-unknown", ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ContentTypeToFileExtension(tt.contentType),
				"ContentTypeToFileExtension(%q)", tt.contentType)
		})
	}
}
