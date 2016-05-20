// ytb
package ytb

import (
	//"github.com/google/google-api-go-client/googleapi/transport"
	//"google.golang.org/api/youtube/v3"
    //"github.com/otium/ytdl"
    "github.com/cavaliercoder/grab"
    "fmt"
	log "youtubedownload/common/youlog"
    "youtubedownload/common/errors"
    "youtubedownload/modeldownload/ytbd/ytdv"
    "time"
    "strings"
	"net/http"
    "net/url"
    "bytes"
)

const (
    cacheCleanTime = 30 * time.Minute
    preVideoSize = 5 * 1024 * 1024
    timeoutCount = 200
)

var (
    file_share_dir = ""
    //developerKey = "AIzaSyDTZj7tbRQscz584zuTAt_xQoIzuxyD9RQ"
    push_server = "http://127.0.0.1:8080"
    push_auth_key = "eOZ6a5R0gl0CZ32HbU7kzst9ASkfjlufK0kGfRZwwroG0WO_wcK1r2msrrH7LUj-VII5taeRQRIPFCYzQJ48oBMB9RqZBMa8v38ZhAzLlpp3suzlxQ-Wb4Qx3f3CdMgk2LW-DdDlVrJotS3K1RUG0aRQMDAOiz3YNZiO4i-c_LiIyw7jDDUObQxnJxWbvsfk"
	vidCache  = make(map[string]Video)
)

type YtbConfig struct {
    PushAddress string
    PushAuthKey string
    FileShareDir string
}

type DownloadRsp struct {
	VideoID string `json:"id"`
    FileName string `json:"filename"`
    FileSize int64 `json:"filesize"`
    Speed int64 `json:"avespeed"`
    LeftTime int64 `json:"lefttime"`
    IsPreVideo bool `json:"bPreVideo"`
	Downloadstatus Format `json:"downloadstatus"`
    PreviewStatus Format `json:"preview"`
    Error errors.Error `json:"error"`
}

type PushServerMsg struct {
    ToPushToken string `json:"to"`
	Data DownloadRsp `json:"data"`
}

type searchInfo struct {
    // The video ID
	ID string `json:"id"`
	// The video title
	Title string `json:"title"`
    // Author of the video
	Author string `json:"author"`
    // Thumbnail url of the video
    ThumbnailURL string `json:"thumbnailurl"`
    // Web page url of the video
    WebPageURL string `json:"webpageurl"`
    // The date the video was published
	DatePublished string `json:"datePublished"`
    // View count of the video
    ViewCount int `json:"viewcount"`
	// Duration of the video
	Duration int64
}

type YtbService struct {
}

func SetYtdConfig(ytbConfig YtbConfig) {
    push_server = ytbConfig.PushAddress
    push_auth_key = ytbConfig.PushAuthKey
    file_share_dir = getCurrentDirectory() + "/" + ytbConfig.FileShareDir
}

func getThumbnailURL(videoID string, quality ytdv.ThumbnailQuality) *url.URL {
	u, _ := url.Parse(fmt.Sprintf("http://img.youtube.com/vi/%s/%s.jpg",
		videoID, quality))
	return u
}

func getThumbnailUrl(uploaderUrl string, videoID string) (strPath string) {
    thumbnailUrl := getThumbnailURL(videoID, ytdv.ThumbnailQualityHigh)
    filePath, err := downloadThumbnailFile(thumbnailUrl.String(), videoID, "/ThumbnailImages")
    
    if err != nil {
        log.Warnning(err.Error())
        return ""
    }
    
    log.Info("Download Thumbnail URL, path is " + filePath)
    upsertPreviewInfoToDatebase(videoID, filePath, uploaderUrl)
    return filePath
}

func getAuthorThumbnailUrl(uploaderUrl string, thumbnailUrl string, videoID string) (strPath string) {
    filePath, err := downloadThumbnailFile(uploaderUrl, videoID, "/UploaderImages")
    
    if err != nil {
        log.Warnning(err.Error())
        return ""
    }
    
    log.Info("Download Author Thumbnail URL, path is " + filePath)
    upsertPreviewInfoToDatebase(videoID, thumbnailUrl, filePath)
    return filePath
}

//Only save one Resoulution
func checkFormatVaild(v ytdv.Format, vs []Format) (bVaild bool) {
    bVaild = true
    
    if len(v.Extension) == 0 || len(v.Resolution) == 0 || len(v.VideoEncoding) == 0 || len(v.AudioEncoding) == 0 || v.AudioBitrate == 0 {
        bVaild = false
        return
    }
    
    for _, item := range vs {
        if item.Resolution == v.Resolution {
            bVaild = false
            return
        }
    }
    
    return
}

func getYoutubeVideoBase(videoID string) Video {
    vid, err := ytdv.GetVideoInfoFromID(videoID)
    
    if err != nil {
        log.Warnning(err.Error())
        return Video{}
    }
    
    videoInfo := Video{
        VideoID:videoID,
        Title:vid.Title,
        Description:vid.Description,
        Keywords:vid.Keywords,
        Author:vid.Author,
        AuthorUrl:vid.AuthorThumbnail,
        DatePublished:vid.DatePublished.Unix(),
        Duration:int64(vid.Duration.Seconds()),
        ViewCount:int64(vid.ViewCount),
        LikeCount:int64(vid.LikeCount),
        DislikeCount:int64(vid.DislikeCount),
    }

    for _, v := range vid.Formats {
        if !checkFormatVaild(v, videoInfo.FormatList) {
            break;
        }

        format := Format{}
        format.Itag = v.Itag
        format.Resolution = v.Resolution
        format.Extension = v.Extension
        videoInfo.FormatList = append(videoInfo.FormatList, format)
    }
    
    return videoInfo
}

/*func (y *YtbService) getStatistics(ids []string)(ret []Video) {
    search := y.svc.Videos.List("id, statistics")
	search = search.Id(strings.Join(ids, ","))

	results, err := search.Do()
	if err != nil {
        log.Warnning("could not retrieve channel list, " + err.Error())
		return nil
	}

	for _, v := range results.Items {
        vid := getYoutubeVideoBase(v.Id)
        vid.LikeCount = (int64)(v.Statistics.LikeCount)
        vid.DislikeCount = (int64)(v.Statistics.DislikeCount)
        vid.ViewCount = (int64)(v.Statistics.ViewCount)
        
        bExist, videoLocal, _, _ := getVideoInfoFromDatebase(v.Id, -1)
        
        if bExist {
            vid.ThumbnailUrl = videoLocal.ThumbnailUrl
            vid.PreviewInfo = videoLocal.PreviewInfo
            
            for index, v := range vid.FormatList {
                for _, item := range videoLocal.FormatList {
                    if item.Itag == v.Itag {
                        v.DownloadUrl = item.DownloadUrl
                        v.Status = item.Status
                        vid.FormatList[index] = v
                        break
                    }
                }
            }
        } else {
            vid.ThumbnailUrl = getThumbnailUrl(v.Id)
        }
        
        vid.recordTime = time.Now().Unix()
        vidCache[v.Id] = vid
        ret = append(ret, vid)
    }

	return ret
}*/

/*func (y *YtbService) getStatisticsV(ids []string, videoList []Video)(ret []Video) {
    search := y.svc.Videos.List("id, statistics")
	search = search.Id(strings.Join(ids, ","))

	results, err := search.Do()
	if err != nil {
        log.Warnning("could not retrieve channel list, " + err.Error())
		return nil
	}

	for _, v := range results.Items {
        vid := getYoutubeVideoBase(v.Id)
        vid.VideoID = v.Id
        vid.LikeCount = (int64)(v.Statistics.LikeCount)
        vid.DislikeCount = (int64)(v.Statistics.DislikeCount)
        vid.ViewCount = (int64)(v.Statistics.ViewCount)
        
        //First Check User downloaded this Video before
        bExist, videoInfo := findVideoInfoInList(v.Id, videoList)
        
        if bExist {
            vid.ThumbnailUrl = videoInfo.ThumbnailUrl
            vid.PreviewInfo = videoInfo.PreviewInfo
            
            for index, v := range vid.FormatList {
                for _, item := range videoInfo.FormatList {
                    if item.Itag == v.Itag {
                        v.DownloadUrl = item.DownloadUrl
                        v.Status = item.Status
                        vid.FormatList[index] = v
                        break
                    }
                }
            }
        } else {//Check if ThumbnailUrl download before
            preViewUrl := searchPreViewInfoFromDatebase(v.Id)

            if len(preViewUrl) > 0 {
                log.Info("Thumbnail URL is downloaded, url is " + preViewUrl)
                vid.ThumbnailUrl = preViewUrl
            } else {
                vid.ThumbnailUrl = getThumbnailUrl(v.Id)
            }
        }
        
        vid.recordTime = time.Now().Unix()
        vidCache[v.Id] = vid
        ret = append(ret, vid)
    }

	return ret
}*/

func (y *YtbService) getSearchList(query string, page int)(ret []searchInfo) {
    searchList, err := ytdv.GetSearchListFromQuery(query, page)
    
    if err != nil {
        log.Warnning("could not get search list, " + err.Error())
		return nil
    }
    
    for _, v := range searchList.PreInfoList {
        searchV := searchInfo{}
        searchV.ID = v.ID
        searchV.Author = v.Author
        searchV.Title = v.Title
        searchV.WebPageURL = v.WebPageURL
        searchV.DatePublished = v.DatePublished
        searchV.Duration = int64(v.Duration)
        searchV.ThumbnailURL = v.ThumbnailURL
        searchV.ViewCount = v.ViewCount
        
        preViewUrl, uploaderUrl := searchPreViewInfoFromDatebase(v.ID)

        if len(preViewUrl) > 0 {
            log.Info("Thumbnail URL is downloaded, url is " + preViewUrl)
            searchV.ThumbnailURL = preViewUrl
        } else {
            searchV.ThumbnailURL = getThumbnailUrl(uploaderUrl, v.ID)
        }
        
        ret = append(ret, searchV)
    }
    
    return
}

func (y *YtbService) getStatisticsV(ids []string, videoList []Video)(ret []Video) {
	for _, v := range ids {
        vid := getYoutubeVideoBase(v)

        //First Check User downloaded this Video before
        bExist, videoInfo := findVideoInfoInList(v, videoList)
        
        if bExist {
            vid.ThumbnailUrl = videoInfo.ThumbnailUrl
            vid.PreviewInfo = videoInfo.PreviewInfo
            
            for index, v := range vid.FormatList {
                for _, item := range videoInfo.FormatList {
                    if item.Itag == v.Itag {
                        v.DownloadUrl = item.DownloadUrl
                        v.Status = item.Status
                        vid.FormatList[index] = v
                        break
                    }
                }
            }
        } else {//Check if ThumbnailUrl download before
            preViewUrl, uploaderUrl := searchPreViewInfoFromDatebase(v)

            if len(preViewUrl) > 0 {
                log.Info("Thumbnail URL is downloaded, url is " + preViewUrl)
                vid.ThumbnailUrl = preViewUrl
            } else {
                preViewUrl = vid.ThumbnailUrl
                vid.ThumbnailUrl = getThumbnailUrl(uploaderUrl, v)
            }
            
            if len(uploaderUrl) > 0 {
                log.Info("Uploader Thumbnail URL is downloaded, url is " + uploaderUrl)
                vid.AuthorUrl = uploaderUrl
            } else {
                vid.AuthorUrl = getAuthorThumbnailUrl(vid.AuthorUrl, preViewUrl, v)
            }
        }
        
        vid.recordTime = time.Now().Unix()
        vidCache[v] = vid
        ret = append(ret, vid)
    }

	return ret
}

func formatBytes(i int64) (result string) {
	switch {
	case i > (1024 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%.02f TB", float64(i)/1024/1024/1024/1024)
	case i > (1024 * 1024 * 1024):
		result = fmt.Sprintf("%.02f GB", float64(i)/1024/1024/1024)
	case i > (1024 * 1024):
		result = fmt.Sprintf("%.02f MB", float64(i)/1024/1024)
	case i > 1024:
		result = fmt.Sprintf("%.02f KB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	result = strings.Trim(result, " ")
	return
}

func sendMessage(downloadReq DownloadReq, response *grab.Response, status int, progress int) {
    pushServerMsg := PushServerMsg{
                            ToPushToken:downloadReq.PushToken,
                            Data:DownloadRsp{
                                VideoID:downloadReq.VideoID,
                                FileName:downloadReq.fileName,
                                FileSize:int64(response.Size),
                                Speed:int64(response.AverageBytesPerSecond()),
                                LeftTime:int64(response.ETA().Sub(time.Now())),
                                IsPreVideo:downloadReq.IsPreVideo,
                                Error:errors.Error{
                                    Id:errors.NoError,
	                                Desc:"",
                                },
                            },
                        }
    
    if downloadReq.IsPreVideo {
        pushServerMsg.Data.PreviewStatus = Format{
                                    Resolution:downloadReq.resolution,
                                    Extension:downloadReq.extension,
                                    Itag:downloadReq.Itag,
                                    Status:status,
                                    Progress:progress,
                                }

       if status == 3 {
           pushServerMsg.Data.PreviewStatus.DownloadUrl = getDownloadURLPath(downloadReq.fileName, "PreVideos")
       }
    } else {
        pushServerMsg.Data.Downloadstatus = Format{
                                    Resolution:downloadReq.resolution,
                                    Extension:downloadReq.extension,
                                    Itag:downloadReq.Itag,
                                    Status:status,
                                    Progress:progress,
                                }
       
       if status == 3 {
           pushServerMsg.Data.Downloadstatus.DownloadUrl = getDownloadURLPath(downloadReq.fileName, downloadReq.resolution)
       }
    }
    
    if status == 3 {
        upsertVideoInfoToDatebase(downloadReq.Token, pushServerMsg)
    }
    
    sendBody := Encode(pushServerMsg)
    go pushToServer(sendBody)
}

func sendPushErrorMsg(downloadReq DownloadReq, err errors.Error) {
    pushServerMsg := PushServerMsg{
                            ToPushToken:downloadReq.PushToken,
                            Data:DownloadRsp{
                                VideoID:downloadReq.VideoID,
                                IsPreVideo:downloadReq.IsPreVideo,
                                Error:err,
                            },
                        }

    sendBody := Encode(pushServerMsg)
    go pushToServer(sendBody)
}

func pushToServer(body []byte) {
    log.Info("Request to Push Server " + string(body))
	httpclient := &http.Client{}
	endpoint := fmt.Sprintf("%s/v1.0/cms", push_server)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
    auth_key := fmt.Sprintf("key=%s", push_auth_key)
    req.Header.Add("Authorization", auth_key)
	resp, err := httpclient.Do(req)
	
    if err != nil {
        log.Warnning("Send Push to server err, " + err.Error())
		return
	}

    log.Info("Responses Status " + resp.Status)
    defer resp.Body.Close()
}

func checkResourcesExists(downloadReq DownloadReq) (bReturn bool) {
    bReturn = false
    bExist, videoLocal, fileName, fileSize := getVideoInfoFromDatebase(downloadReq.VideoID, int32(downloadReq.Itag))
    
    if bExist {
        if downloadReq.IsPreVideo && videoLocal.PreviewInfo.Itag > 0 && videoLocal.PreviewInfo.Status == 3 {
            bReturn = true
        } else if downloadReq.IsPreVideo && len(videoLocal.FormatList) > 0 && videoLocal.FormatList[0].Status == 3 {
            bReturn = true
        }
        
        if bReturn {
            pushServerMsg := PushServerMsg{
                            ToPushToken:downloadReq.PushToken,
                            Data:DownloadRsp{
                                VideoID:videoLocal.VideoID,
                                FileName:fileName,
                                FileSize:fileSize,
                                Speed:0,
                                LeftTime:0,
                                IsPreVideo:downloadReq.IsPreVideo,
                            },
                        }
           
            if downloadReq.IsPreVideo {
                pushServerMsg.Data.PreviewStatus = Format{
                                    DownloadUrl:videoLocal.PreviewInfo.DownloadUrl,
                                    Resolution:videoLocal.PreviewInfo.Resolution,
                                    Extension:videoLocal.PreviewInfo.Extension,
                                    Itag:videoLocal.PreviewInfo.Itag,
                                    Status:videoLocal.PreviewInfo.Status,
                                    Progress:videoLocal.PreviewInfo.Progress,
                                }
                
                log.Info("Preview Video is downloaded, so update video cache info, videoID is " + downloadReq.VideoID)
                updateVideoInfoToCache(pushServerMsg)
             } else {
                pushServerMsg.Data.Downloadstatus = Format{
                                    DownloadUrl:videoLocal.FormatList[0].DownloadUrl,
                                    Resolution:videoLocal.FormatList[0].Resolution,
                                    Extension:videoLocal.FormatList[0].Extension,
                                    Itag:videoLocal.FormatList[0].Itag,
                                    Status:videoLocal.FormatList[0].Status,
                                    Progress:videoLocal.FormatList[0].Progress,
                                }
                
                log.Info("Video is downloaded, so update datebase and video cache info, videoID is " + downloadReq.VideoID)
                upsertVideoInfoToDatebase(downloadReq.Token, pushServerMsg)
            }
    
            sendBody := Encode(pushServerMsg)
            go pushToServer(sendBody)
        }
    }

    return bReturn
}

func startDownload(downloadReq DownloadReq) {
    vidInfo, err := ytdv.GetVideoInfoFromID(downloadReq.VideoID)
    
    if err != nil {
        log.Warnning("Download error, get videoinfo fatal!!!")
        errRsp := errors.NewError(errors.HttpError, "Network Error" + "Download error, get videoinfo fatal, " + err.Error())
        sendPushErrorMsg(downloadReq, *errRsp)
		return
	}
    
    formats := ytdv.FormatList{}
   
    for _, v := range vidInfo.Formats {
        if len(v.Extension) == 0 || len(v.Resolution) == 0 || len(v.VideoEncoding) == 0 || len(v.AudioEncoding) == 0 || v.AudioBitrate == 0 {
            break;
        }
        
        formats = append(formats, v)
    }
    
    var downloadFormat ytdv.Format
    
    if downloadReq.IsPreVideo {
        for _, item := range formats {
		    if item.Resolution == "360p" {
			    downloadFormat = item
			    break
		    }
	    }
    } else {
        for _, item := range formats {
		    if item.Itag == downloadReq.Itag {
			    downloadFormat = item
			    break
		    }
	    }
    }
    
    if downloadFormat.Itag <= 0 {
        log.Info("Download format is invalid, use the worest format!")
        downloadFormat = formats.Worst(ytdv.FormatResolutionKey)[0]
    }

    downloadURL, err := vidInfo.GetDownloadURL(downloadFormat)

	if err != nil {
        log.Warnning("Download error, get download url fatal!!!")
        errRsp := errors.NewError(errors.HttpError, "Network Error" + "Download error, get download url fatal, " + err.Error())
        sendPushErrorMsg(downloadReq, *errRsp)
		return
	}
    
    dirResolutionPath := ""
    
    if downloadReq.IsPreVideo {
        dirResolutionPath = generateDownloadFilePath("/PreVideos")
    } else {
        dirResolutionPath = generateDownloadFilePath("/" + downloadFormat.Resolution)
    }
    
    downloadReq.Itag = downloadFormat.Itag
    downloadReq.url = downloadURL.String()
    downloadReq.fileName = vidInfo.ID + "." + downloadFormat.Extension
    downloadReq.filePath = dirResolutionPath
    downloadReq.resolution = downloadFormat.Resolution
    downloadReq.extension = downloadFormat.Extension
    
    //Insert a message in database
    pushServerMsg := PushServerMsg{
                            ToPushToken:downloadReq.PushToken,
                            Data:DownloadRsp{
                                VideoID:downloadReq.VideoID,
                                FileName:downloadReq.fileName,
                                IsPreVideo:downloadReq.IsPreVideo,
                            },
                        }
           
    if downloadReq.IsPreVideo {
        pushServerMsg.Data.PreviewStatus = Format{
            Resolution:downloadReq.resolution,
            Extension:downloadReq.extension,
            Itag:downloadReq.Itag,
            Status:1,
         }
    } else {
        pushServerMsg.Data.Downloadstatus = Format{
            Resolution:downloadReq.resolution,
            Extension:downloadReq.extension,
            Itag:downloadReq.Itag,
            Status:1,
         }
    }
             
    upsertVideoInfoToDatebase(downloadReq.Token, pushServerMsg)
    downloadYoutubeVideo(downloadReq)
}

func downloadYoutubeVideo(downloadReq DownloadReq) {
	// create a custom client
	client := grab.NewClient()
	client.UserAgent = ""

	// create requests from command arguments
	reqs := make([]*grab.Request, 0)
    req, err := grab.NewRequest(downloadReq.url)
    
    if err != nil {
        log.Warnning("Download error " + err.Error())
		//fmt.Fprintf(os.Stderr, "%v\n", err)
        errRsp := errors.NewError(errors.HttpError, "Download error " + err.Error())
        sendPushErrorMsg(downloadReq, *errRsp)
		return
	}
		
    req.Filename = downloadReq.filePath + "/" + downloadReq.fileName
    
    if downloadReq.IsPreVideo {
        req.HTTPRequest.Header.Set("Range", "bytes=0-" + fmt.Sprintf("%d", preVideoSize))
        log.Info("Download PreVideo " + "bytes=0-" + fmt.Sprintf("%d", preVideoSize))
    }
    
	reqs = append(reqs, req)

	/*for _, item := range fileList {
		req, err := grab.NewRequest(item.url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		req.Filename = item.fileName
		reqs = append(reqs, req)
	}*/
    
	// start file downloads, 3 at a time
    log.Info(fmt.Sprintf(">>>>>>>>>>>>>>>>>>>Downloading file name is %s, file path is %s\n\n", downloadReq.fileName, req.Filename))
	respch := client.DoBatch(3, reqs...)

	// start a ticker to update progress every 3s
	t := time.NewTicker(3000 * time.Millisecond)

	// monitor downloads
	completed := 0
	inProgress := 0
    reConnect := 0
    var transferred uint64
	responses := make([]*grab.Response, 0)
	for completed < len(reqs) {
		select {
		case resp := <-respch:
			// a new response has been received and has started downloading
			// (nil is received once, when the channel is closed by grab)
			if resp != nil {
				responses = append(responses, resp)
			}

		case <-t.C:
			// clear lines
			if inProgress > 0 {
                log.Info(fmt.Sprintf("\033[%dA\033[K", inProgress))
			}

			// update completed downloads
			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					// print final result
					if resp.Error != nil {
                        log.Warnning(fmt.Sprintf("############# [Error] downloading %s: %v\n", resp.Request.URL(), resp.Error))
                        errRsp := errors.NewError(errors.HttpError, "Download error " + resp.Error.Error())
                        sendPushErrorMsg(downloadReq, *errRsp)
					} else {
                        log.Info(fmt.Sprintf("%s downloading completely, >>>>>>>>>>>>>>>>>>> %s / %s, speed is %.2f kb/s, total time is %s\n", resp.Filename, formatBytes(int64(resp.BytesTransferred())),
                                formatBytes(int64(resp.Size)), resp.AverageBytesPerSecond() / 1024, resp.Duration().String()))
                         sendMessage(downloadReq, resp, 3, 1000)
					}

					// mark completed
					responses[i] = nil
					completed++
				}
			}

			// update downloads in progress
			inProgress = 0
			for _, resp := range responses {
				if resp != nil {
                    if resp.Error != nil {
                        log.Warnning(fmt.Sprintf("############# [Error] downloading %s: %v\n", resp.Request.URL(), resp.Error))
                        errRsp := errors.NewError(errors.InternalError, "Download error " + resp.Error.Error())
                        sendPushErrorMsg(downloadReq, *errRsp)
                        client.CancelRequest(resp.Request)
                        
                        // mark completed
					    completed++
                        break
                    } else {
                        //Check network timeout if over 1 minute, cancel request.
                        if reConnect > timeoutCount {
                            reConnect = 0
                            transferred = 0
                            log.Warnning(fmt.Sprintf("############# [Error] downloading %s timeout, video id is %s, cancel download request", resp.Filename, downloadReq.VideoID))
                            errRsp := errors.NewError(errors.HttpError, "Download Timeout!!!")
                            sendPushErrorMsg(downloadReq, *errRsp)
                            client.CancelRequest(resp.Request)
                        
                            // mark completed
					        completed++
                            break
                        }
                        
                        if transferred < resp.BytesTransferred() {
                            reConnect = 0
                            transferred = resp.BytesTransferred()
                        } else {
                            reConnect++
                        }
                        
                        //Check Resume is supported
                        if inProgress == 0 && transferred > 0 && transferred < preVideoSize && downloadReq.IsPreVideo {
                            resp.Request.HTTPRequest.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", transferred, preVideoSize))
                            log.Info("Location resume Download PreVideo, set Request Header Range " + fmt.Sprintf("bytes=%d-%d", transferred, preVideoSize))
                        }
                        
					    inProgress++
                        log.Info(fmt.Sprintf("Downloading %s >>>>>>>>>>>>>>>>>>> %s / %s progress (%.2f%%)\033[K, speed is %.2f kb/s, estimated time %s\n\n", resp.Filename, formatBytes(int64(resp.BytesTransferred())),
                                formatBytes(int64(resp.Size)), 100*resp.Progress(), resp.AverageBytesPerSecond() / 1024, resp.ETA().Sub(time.Now()).String()))
                        sendMessage(downloadReq, resp, 1, int(1000*resp.Progress()))
                    }
				}
			}
		}
	}

	t.Stop()
    log.Info(fmt.Sprintf("%s download request end.\n", downloadReq.fileName))
}

