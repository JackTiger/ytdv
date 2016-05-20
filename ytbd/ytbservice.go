// ytbmsg
package ytb

import (
	log "youtubedownload/common/youlog"
    "youtubedownload/common/errors"
)

const (
    TopicRequest = "Topic_Request"
    ChannelRequest = "Channel_To_Ytbd"
	HandleDownload  = "download"
	HandlePreview    = "preview"
)

type NsqRequest struct {
    HandleOperator string `json:"handleoperator"`
    VideoID string `json:"id"`
    Token string `json:"access_token"`
    PushToken string `json:"push_token"`
    Itag   int    `json:"itag"`
}

type SearchReq struct {
	KeyWord string `json:"keyword"`
	Page int `json:"page"`
    Token string `json:"access_token"`
}

type SearchRsp struct {
    Page int `json:"page"`
    SearchList []searchInfo `json:"preview_contents"`
}

type GetTasksReq struct {
	PageToken int64 `json:"next_token"`
    Token string `json:"access_token"`
    PageCount int64 `json:"pagecount"`
}

type GetTasksRsp struct {
    PageToken int64 `json:"next_token"`
    VideoList []Video `json:"videos_content"`
}

type GetVideoReq struct {
    VideoID string `json:"id"`
    Token string `json:"access_token"`
}

type DeleteReq struct {
    VideoID string `json:"id"`
    Itag          int    `json:"itag"`
    Token string `json:"access_token"`
}

type Format struct {
    DownloadUrl string `json:"download_url"`
	Resolution    string `json:"resolution"`
    Extension string `json:"extension"`
    Itag          int    `json:"itag"`
    Status int `json:"status"`
    Progress int `json:"progress"`
}

type Video struct {
    VideoID string `json:"id"`
    ThumbnailUrl string `json:"preview_url"`
    Title string `json:"title"`
    Description string `json:"description"`
    Keywords []string `json:"keywords"`
    Author string `json:"author"`
    AuthorUrl string `json:"authorurl"`
    DatePublished int64 `json:"datepublished"`
    Duration int64 `json:"duration"`
    ViewCount int64 `json:"viewcount"`
    LikeCount int64 `json:"likecount"`
    DislikeCount int64 `json:"dislikecount"`
    FormatList []Format `json:"formats"`
    PreviewInfo Format `json:"preview"`
    recordTime int64
}

type DownloadReq struct {
    VideoID string `json:"id"`
    Token string `json:"access_token"`
    PushToken string `json:"push_token"`
    Itag    int    `json:"itag"`
    IsPreVideo bool `json:"is_prevideo"`
	url string
	fileName string
    filePath string
    resolution string
    extension string
}

func (y *YtbService) SearchFromKeyword(searchReq SearchReq) (searchRsp SearchRsp, errRsp *errors.Error) {
    nPage := 1
    keyWord := "Google"
    errRsp = nil
    
    if len(searchReq.KeyWord) > 0 {
        keyWord = searchReq.KeyWord
    }
    
    if searchReq.Page > 0 {
        nPage = searchReq.Page
    }
    
    log.Info(string(Encode(searchReq)))
    searchRsp.Page = nPage
    
    results := y.getSearchList(keyWord, nPage)
    searchRsp.SearchList = results
    log.Info(string(Encode(searchRsp)))
    return
}

func (y *YtbService) GetTaskList(getTasksReq GetTasksReq) (getTasksRsp GetTasksRsp, errRsp *errors.Error) {
    if getTasksReq.PageCount == 0 {
        getTasksReq.PageCount = 5
    }
    
    errRsp = nil
    log.Info(string(Encode(getTasksReq)))
    bExist, videoLocalList, ids, pageToken := getUserTaskListFromDatebase(getTasksReq)
    
    if bExist {
        var toFetch []string
        getTasksRsp.PageToken = pageToken
	    ret := retrieveCache(ids, vidCache, &toFetch, getTasksReq.Token)
	
        if 0 == len(toFetch) {
            getTasksRsp.VideoList = ret
            log.Info(string(Encode(getTasksRsp)))
            return
	    }
        
        retServer := y.getStatisticsV(toFetch, videoLocalList)
        
        if retServer == nil {
            errRsp = errors.NewError(errors.HttpError, "Network Error, could not retrieve video list")
		    return
        }
    
        for index, v := range ret {
            if len(v.Title) > 0 || len(v.Description) > 0 {
                break
            } else {
                bExist, videoInfo := findVideoInfoInList(v.VideoID, retServer)
            
                if bExist {
                    ret[index] = videoInfo
                }
            }
        }
    
        getTasksRsp.VideoList = ret
    }
    
    log.Info(string(Encode(getTasksRsp)))
    return
}

func (y *YtbService) GetVideoDetail(getVideoReq GetVideoReq) (videoInfo Video, errRsp *errors.Error) {
    log.Info(string(Encode(getVideoReq)))
    errRsp = nil
    
    if len(getVideoReq.VideoID) == 0 {
        errRsp = errors.NewError(errors.InvalidError, "Video Id is empty!!!")
        return
    }
    
    ids := []string{getVideoReq.VideoID}
    
    var toFetch []string
	ret := retrieveCache(ids, vidCache, &toFetch, getVideoReq.Token)
	
    if 0 == len(toFetch) {
        videoInfo = ret[0]
        log.Info(string(Encode(videoInfo)))
        return
	}
    
    videoUserList := []Video{}
    
    if len(getVideoReq.Token) > 0 {
        //Get all data record of user download list
        _, videoLocalList, _, _ := getUserTaskListFromDatebase(GetTasksReq {
            PageToken:0,
            Token:getVideoReq.Token,
            PageCount:0,
        })
        
        if videoLocalList != nil {
            videoUserList = videoLocalList
        }
    }
    
    ret = y.getStatisticsV(toFetch, videoUserList)
    
    if ret == nil {
        errRsp = errors.NewError(errors.HttpError, "Network Error, could not retrieve video list")
		return
    }
        
    videoInfo = ret[0]
    log.Info(string(Encode(videoInfo)))
    
    return
}

func (y *YtbService) DeleteVideo(deleteReq DeleteReq) (errRsp *errors.Error) {
    log.Info(string(Encode(deleteReq)))
    errRsp = nil
    err := deleteVideoInfoToDatebase(deleteReq)
    
    if err != nil {
        log.Warnning("Delete error " + err.Error())
        errRsp = errors.NewError(errors.DbError, "Delete Error" + err.Error())
		return
	}
    
    log.Info("Delete Successfully!")
    return 
}

func DownloadVideo(downloadReq DownloadReq) {
    log.Info(string(Encode(downloadReq)))
    //Check DataBase if exist, direct push to client
    if bExist := checkResourcesExists(downloadReq); !bExist {
        startDownload(downloadReq)
    }
    
    //For test
    /*
    pushServerMsg := PushServerMsg{
                            ToPushToken:downloadReq.Token,
                            Data:DownloadRsp{
                                VideoID:downloadReq.VideoID,
                            },
                        }
    
    upsertVideoInfoToDatebase(pushServerMsg)*/
}