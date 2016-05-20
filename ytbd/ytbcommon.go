// ytbcommon
package ytb

import (
    "time"
	"net/http"
    "net"
    "errors"
	"io/ioutil"
    "path/filepath"
    "encoding/json"
	"strings"
    "os"
    "fmt"
    log "youtubedownload/common/youlog"
)

func Encode(voidHandle interface{}) []byte {
	buf, err := json.Marshal(voidHandle)
	if err != nil {
		return buf
	}

	return buf
}

func Decode(mesData []byte, voidHandle interface{}){
    err := json.Unmarshal(mesData, voidHandle)
    
	if err != nil {
		return
	}
}

func UpdateCache(videoInfo Video) {
    log.Info("Update Video download state is " + string(Encode(videoInfo)))
    
    if v, ok := vidCache[videoInfo.VideoID]; ok {
        log.Info("Cache Data before update is " + string(Encode(v)))
        if videoInfo.PreviewInfo.Itag > 0 {
            if len(videoInfo.PreviewInfo.DownloadUrl) > 0 {
                v.PreviewInfo.DownloadUrl = videoInfo.PreviewInfo.DownloadUrl
            }
            
            v.PreviewInfo.Status = videoInfo.PreviewInfo.Status
            v.PreviewInfo.Extension = videoInfo.PreviewInfo.Extension
            v.PreviewInfo.Resolution = videoInfo.PreviewInfo.Resolution
            v.PreviewInfo.Itag = videoInfo.PreviewInfo.Itag
        }
        
        for _, format := range videoInfo.FormatList {
            for index, item := range v.FormatList {
                if item.Itag == format.Itag {
                    if len(format.DownloadUrl) > 0 {
                        item.DownloadUrl = format.DownloadUrl
                    }
                    
                    item.Extension = format.Extension
                    item.Resolution = format.Resolution
                    item.Status = format.Status
                    v.FormatList[index] = item
                    break
                }
            }
        }
        
        log.Info("Cache Data after update is " + string(Encode(v)))
        vidCache[videoInfo.VideoID] = v
    }
}

func retrieveCache(ids []string, cache map[string]Video, unmatched *[]string, token string) (ret []Video) {
	for _, id := range ids {
		if v, ok := cache[id]; ok {
            if (time.Duration)(time.Now().Unix() - v.recordTime) > cacheCleanTime {
                delete(cache,id)
                *unmatched = append(*unmatched, id)
                ret = append(ret, Video{
                    VideoID:id,
                })
            } else {
                if len(token) > 0 {
                    ret = append(ret, v)
                } else {
                    for index, item := range v.FormatList {
                        item.Status = 0
                        item.DownloadUrl = ""
                        v.FormatList[index] = item
                    }
    
                    v.PreviewInfo.DownloadUrl = ""
                    v.PreviewInfo.Status = 0
                    ret = append(ret, v)
                }
            }
		} else {
			*unmatched = append(*unmatched, id)
            ret = append(ret, Video{
                VideoID:id,
            })
		}
	}
	return
}

func findVideoInfoInList(ID string, list []Video) (bool, Video) {
    bExist := false
    videoInfo := Video{}
    
    for _, item := range list {
        if item.VideoID == ID {
            bExist = true
            videoInfo = item
            break
        }
    }
    
    return bExist, videoInfo
}

// Fetch function
func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New("Failed to fetch " + url)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("Failed to parse " + url)
	}
	return body, nil
}

func isDirExists(path string) bool {
    fi, err := os.Stat(path)
 
    if err != nil {
        return os.IsExist(err)
    } else {
        return fi.IsDir()
    }
 
    panic("not reached")
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func generateDownloadFilePath(subDir string) string {
    fileRoot := file_share_dir
    
    if !isDirExists(fileRoot + subDir) {
        os.Mkdir(fileRoot + subDir, 0777)
    }
    
    return fileRoot + subDir
}

func downloadThumbnailFile(url string, fileName string, fileDir string) (string, error){
    bytes, err := fetch(url)
    
    if err != nil {
        return "", err
    }
    
    fileRoot := file_share_dir
    dirResolutionPath :=  fileRoot + fileDir
    
    if !isDirExists(dirResolutionPath) {
        os.Mkdir(dirResolutionPath, 0777)
    }
    
    filePath := dirResolutionPath + "/" + fileName + ".png"
    err = ioutil.WriteFile(filePath, bytes, 0644)
    
    if err != nil {
        return "", err
    }
    
    filePath = "http://" + getLocalAddr() + ":3000" + fileDir + "/" + fileName + ".png"
    return filePath, nil
}

func getThumbnailURLPath(fileName string) string{
    return "http://" + getLocalAddr() + ":3000" + "/ThumbnailImages/" + fileName + ".png"
}

func getDownloadURLPath(fileName string, subPath string) string{
    return "http://" + getLocalAddr() + ":3000/" + subPath + "/" + fileName
}

func getLocalAddr() string { //Get ip
    /*conn, err := net.Dial("udp", "baidu.com:80")
    if err != nil {
        fmt.Println(err.Error())
        return "Erorr"
    }
    defer conn.Close()
    return strings.Split(conn.LocalAddr().String(), ":")[0]*/
    
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
    
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
			}
		}
	}
    
    return ""
}
