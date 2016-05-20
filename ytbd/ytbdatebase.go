// ytbdatebase
package ytb

import (
    "fmt"
    "golang.org/x/net/context"
	"google.golang.org/grpc"
	//"google.golang.org/grpc/grpclog"
    pb "youtubedownload/routeguide/db"
    pbytb "youtubedownload/routeguide/ytb"
    log "youtubedownload/common/youlog"
)

var grpcConnect *grpc.ClientConn
var grpcClient pb.RouteGuideDBClient
var grpcConnectYtb *grpc.ClientConn
var grpcClientYtb pbytb.GrpcytbClient

type taskdatebase struct {
    imageURL string
    formatList []Format
    previewInfo []Format
}

func InitYtdDatebase(cc *grpc.ClientConn) {
    grpcConnect = cc
    grpcClient = pb.NewRouteGuideDBClient(grpcConnect)
}

func InitYtdService(ccYtb *grpc.ClientConn) {
    grpcConnectYtb = ccYtb
    grpcClientYtb = pbytb.NewGrpcytbClient(grpcConnectYtb)
}

func isExistsInArray(arr []interface {}, key interface {}) bool {
    bExists := false
    
    for _, item := range arr {
        if item == key {
            bExists = true
            break
        }
    }
    
    return bExists
}

func getUserTaskListFromDatebase(getTasksReq GetTasksReq)(bool, []Video, []string, int64) {
    rpcDownloadStateData := pb.RpcDownloadStateData {
	    PageToken:getTasksReq.PageToken,
        PageCount:int32(getTasksReq.PageCount),
        AccessToken:getTasksReq.Token,
    }
    
    log.Info(string(Encode(rpcDownloadStateData)))
    ret, err := grpcClient.GetDownloadStateDataHandler(context.Background(), &rpcDownloadStateData)
    
    if err != nil {
        log.Warnning(fmt.Sprintf("%v.getUserTaskListFromDatebase(_) = _, %v: ", grpcClient, err))
        return false, nil, nil, 0
	}
    
    if ret.Error.ErrCode != 0 {
        log.Warnning("Error description is " + ret.Error.ErrDesc)
        return false, nil, nil, 0
    }
    
    if len(ret.Item) == 0 {
        return false, nil, nil, 0
    }
    
    var pageToken int64
    var ids []interface {}
    var tasksMap map[string]taskdatebase
    tasksMap = make(map[string]taskdatebase)
    
    for _, item := range ret.Item {
        if !isExistsInArray(ids, item.VideoID) {
            ids = append(ids, item.VideoID)
        }

        if tasksItem, ok := tasksMap[item.VideoID]; ok {
            if item.DownloadItag > 0 {
                tasksItem.formatList = append(tasksItem.formatList, Format {
                    DownloadUrl:item.DownloadURL,
	                Resolution:item.DownloadResolution,
                    Extension:item.DownloadExt,
                    Itag:int(item.DownloadItag),
                    Status:int(item.DownloadStatus),
                })
            }
            
            if item.PreviewItag > 0 {
                tasksItem.previewInfo = append(tasksItem.previewInfo, Format {
                    DownloadUrl:item.PreviewURL,
	                Resolution:item.PreviewResolution,
                    Extension:item.PreviewExt,
                    Itag:int(item.PreviewItag),
                    Status:int(item.PreviewStatus),
                })
            }
            
            tasksItem.imageURL = item.ImageURL
            tasksMap[item.VideoID] = tasksItem
        }else {
            taskItem := taskdatebase{}
            
            if item.DownloadItag > 0 {
                taskItem.formatList = append(taskItem.formatList, Format {
                    DownloadUrl:item.DownloadURL,
	                Resolution:item.DownloadResolution,
                    Extension:item.DownloadExt,
                    Itag:int(item.DownloadItag),
                    Status:int(item.DownloadStatus),
                })
            }
            
            if item.PreviewItag > 0 {
                taskItem.previewInfo = append(taskItem.previewInfo, Format {
                    DownloadUrl:item.PreviewURL,
	                Resolution:item.PreviewResolution,
                    Extension:item.PreviewExt,
                    Itag:int(item.PreviewItag),
                    Status:int(item.PreviewStatus),
                })
            }
            
            taskItem.imageURL = item.ImageURL
            tasksMap[item.VideoID] = taskItem
        }
    }
    
    videoList := []Video{}
    var videoIDs []string
    
    for _, item := range ids {
        video := Video{}
        video.VideoID = item.(string)
        video.ThumbnailUrl = tasksMap[item.(string)].imageURL
        
        if len(tasksMap[item.(string)].formatList) > 0 {
            video.FormatList = tasksMap[item.(string)].formatList
        }
        
        if len(tasksMap[item.(string)].previewInfo) > 0 {
            video.PreviewInfo = tasksMap[item.(string)].previewInfo[0]
        }
        
        videoList = append(videoList, video)
        videoIDs = append(videoIDs, item.(string))
    }
    
    log.Info(string(Encode(videoList)))
    pageToken = ret.Item[len(ret.Item) - 1].Createtime
    return true, videoList, videoIDs, pageToken
}

func getVideoInfoFromDatebase(videoID string, itag int32)(bool, Video, string, int64) {
    rpcSearchVideoInfo := pb.RpcSearchVideoInfo {
	    VideoID:videoID,
        Itag:itag,
    }
    
    log.Info(string(Encode(rpcSearchVideoInfo)))
    ret, err := grpcClient.GetVideoInfobyVideoIDHandler(context.Background(), &rpcSearchVideoInfo)
    
    if err != nil {
        log.Warnning(fmt.Sprintf("%v.getVideoInfoFromDatebase(_) = _, %v: ", grpcClient, err))
        return false, Video{}, "", 0
	}
    
    if ret.Error.ErrCode != 0 {
        log.Warnning("Error description is " + ret.Error.ErrDesc)
        return false, Video{}, "", 0
    }
    
    if len(ret.Item) == 0 {
        log.Info("Local Database not find this video log " + videoID)
        return false, Video{}, "", 0
    }
    
    var tasksMap map[string]taskdatebase
    tasksMap = make(map[string]taskdatebase)
    
    for _, item := range ret.Item {
         log.Info(string(Encode(item)))
        if tasksItem, ok := tasksMap[item.VideoID]; ok {
            if item.DownloadItag > 0 {
                tasksItem.formatList = append(tasksItem.formatList, Format {
                    DownloadUrl:item.DownloadURL,
	                Resolution:item.DownloadResolution,
                    Extension:item.DownloadExt,
                    Itag:int(item.DownloadItag),
                    Status:int(item.DownloadStatus),
                })
            }
            
            if item.PreviewItag > 0 {
                tasksItem.previewInfo = append(tasksItem.previewInfo, Format {
                    DownloadUrl:item.PreviewURL,
	                Resolution:item.PreviewResolution,
                    Extension:item.PreviewExt,
                    Itag:int(item.PreviewItag),
                    Status:int(item.PreviewStatus),
                })
            }
            
            tasksItem.imageURL = item.ImageURL
            tasksMap[item.VideoID] = tasksItem
        }else {
            taskItem := taskdatebase{}
            
            if item.DownloadItag > 0 {
                taskItem.formatList = append(taskItem.formatList, Format {
                    DownloadUrl:item.DownloadURL,
	                Resolution:item.DownloadResolution,
                    Extension:item.DownloadExt,
                    Itag:int(item.DownloadItag),
                    Status:int(item.DownloadStatus),
                })
            }
            
            if item.PreviewItag > 0 {
                taskItem.previewInfo = append(taskItem.previewInfo, Format {
                    DownloadUrl:item.PreviewURL,
	                Resolution:item.PreviewResolution,
                    Extension:item.PreviewExt,
                    Itag:int(item.PreviewItag),
                    Status:int(item.PreviewStatus),
                })
            }
            
            taskItem.imageURL = item.ImageURL
            tasksMap[item.VideoID] = taskItem
        }
    }
    
    video := Video{}
    video.VideoID = videoID
    video.ThumbnailUrl = tasksMap[videoID].imageURL
    
    if len(tasksMap[videoID].formatList) > 0 {
        video.FormatList = tasksMap[videoID].formatList
    }
    
    if len(tasksMap[videoID].previewInfo) > 0 {
        video.PreviewInfo = tasksMap[videoID].previewInfo[0]
    }
    
    fileName := ret.Item[0].FileName
    fileSize := ret.Item[0].FileSize
    log.Info(string(Encode(video)))
    
    return true, video, fileName, fileSize
}

func updateVideoInfoToCache(pushServerMsg PushServerMsg) {
    if grpcClientYtb != nil {
        rpcVideo := pbytb.RpcVideo{}
        rpcVideo.VideoID = pushServerMsg.Data.VideoID

        rpcFormats := []*pbytb.RpcFormat{}
        rpcFormat := pbytb.RpcFormat{}
        rpcFormat.DownloadUrl = pushServerMsg.Data.Downloadstatus.DownloadUrl
        rpcFormat.Extension = pushServerMsg.Data.Downloadstatus.Extension
        rpcFormat.Resolution = pushServerMsg.Data.Downloadstatus.Resolution
        rpcFormat.Itag = (int32)(pushServerMsg.Data.Downloadstatus.Itag)
        rpcFormat.Status = (int32)(pushServerMsg.Data.Downloadstatus.Status)
        rpcFormats = append(rpcFormats, &rpcFormat)
        rpcVideo.FormatList = rpcFormats
    
        rpcPreview := pbytb.RpcFormat{}
        rpcPreview.DownloadUrl = pushServerMsg.Data.PreviewStatus.DownloadUrl
        rpcPreview.Extension = pushServerMsg.Data.PreviewStatus.Extension
        rpcPreview.Resolution = pushServerMsg.Data.PreviewStatus.Resolution
        rpcPreview.Itag = (int32)(pushServerMsg.Data.PreviewStatus.Itag)
        rpcPreview.Status = (int32)(pushServerMsg.Data.PreviewStatus.Status)
        rpcVideo.PreviewInfo = &rpcPreview

        log.Info(string(Encode(rpcVideo)))
        rpcErrorRetData, err := grpcClientYtb.UpdateVideoCache(context.Background(), &rpcVideo)
        
        if err != nil {
            log.Warnning(fmt.Sprintf("%v.updateVideoInfoToCache(_) = _, %v: ", grpcClientYtb, err))
	    }
        
        if rpcErrorRetData.ErrCode != 0 {
            log.Warnning("Error description is " + rpcErrorRetData.ErrDesc)
        } else {
            log.Info("Update Cache successfully!!!")
        }
    }
}

 func upsertVideoInfoToDatebase(accessToken string, pushServerMsg PushServerMsg) {
     rpcUpSertVideoInfo := pb.RpcUpSertVideoInfo{ 
         Token:accessToken,
         Item:&pb.RpcDownloadStateRetData{
             FileName:pushServerMsg.Data.FileName,
             FileSize:pushServerMsg.Data.FileSize,
             VideoID:pushServerMsg.Data.VideoID,
             ImageURL:getThumbnailURLPath(pushServerMsg.Data.VideoID),
             DownloadURL:pushServerMsg.Data.Downloadstatus.DownloadUrl,
             DownloadItag:int32(pushServerMsg.Data.Downloadstatus.Itag),
             DownloadResolution:pushServerMsg.Data.Downloadstatus.Resolution,
             DownloadExt:pushServerMsg.Data.Downloadstatus.Extension,
             DownloadStatus:int32(pushServerMsg.Data.Downloadstatus.Status),
             PreviewURL:pushServerMsg.Data.PreviewStatus.DownloadUrl,
             PreviewItag:int32(pushServerMsg.Data.PreviewStatus.Itag),
             PreviewResolution:pushServerMsg.Data.PreviewStatus.Resolution,
             PreviewExt:pushServerMsg.Data.PreviewStatus.Extension,
             PreviewStatus:int32(pushServerMsg.Data.PreviewStatus.Status),
         },
     }
     
     log.Info(string(Encode(rpcUpSertVideoInfo)))
     rpcErrorRetData, err := grpcClient.UpsertVideoInfoHandler(context.Background(), &rpcUpSertVideoInfo)
     
    if err != nil {
        log.Warnning(fmt.Sprintf("%v.upsertVideoInfoToDatebase(_) = _, %v: ", grpcClient, err))
	}
    
    updateVideoInfoToCache(pushServerMsg)
    
    if rpcErrorRetData.ErrCode != 0 {
        log.Warnning("Error description is " + rpcErrorRetData.ErrDesc)
    } else {
        log.Info("Update to datebase successfully!!!")
    }
 }

 func deleteVideoInfoToDatebase(deleteReq DeleteReq) error {
     rpcDelVideoInfo := pb.RpcDeleteVideoInfo {
         Token:deleteReq.Token,
         Itag:int32(deleteReq.Itag),
         VideoID:deleteReq.VideoID,
     }
     
     log.Info(string(Encode(rpcDelVideoInfo)))
     rpcErrorRetData, err := grpcClient.DeleteVideoInfoHandler(context.Background(), &rpcDelVideoInfo)
     
    if err != nil {
        log.Warnning(fmt.Sprintf("%v.deleteVideoInfoToDatebase(_) = _, %v: ", grpcClient, err))
        return err
	}
    
    if rpcErrorRetData.ErrCode != 0 {
        log.Warnning("Error description is " + rpcErrorRetData.ErrDesc)
    } else {
        log.Info("Delete to datebase successfully!!!")
    } 
    
    return nil
}

func upsertPreviewInfoToDatebase(videoID string, preViewUrl string, authorUrl string) {
    rpcUpSertPreviewInfo := pb.RpcUpsertPreviewInfo{
        VideoID:videoID,
        PreviewURL:preViewUrl,
        UploaderURL:authorUrl,
    }

     log.Info(string(Encode(rpcUpSertPreviewInfo)))
     rpcErrorRetData, err := grpcClient.UpsertPreviewDataHandler(context.Background(), &rpcUpSertPreviewInfo)
     
     if err != nil {
        log.Warnning(fmt.Sprintf("%v.upsertPreviewInfoToDatebase(_) = _, %v: ", grpcClient, err))
	}
    
    if rpcErrorRetData.ErrCode != 0 {
        log.Warnning("Error description is " + rpcErrorRetData.ErrDesc)
    } else {
        log.Info("Update to datebase successfully!!!")
    } 
}

func searchPreViewInfoFromDatebase(videoID string) (preViewUrl string, uploaderUrl string) {
    rpcVideoIDDataInfo := pb.RpcVideoIDDataInfo{
        VideoID:videoID,
    }

     preViewUrl = ""
     uploaderUrl = ""
     log.Info(string(Encode(rpcVideoIDDataInfo)))
     rpcPreviewRetData, err := grpcClient.SearchPreviewDataHandler(context.Background(), &rpcVideoIDDataInfo)
     
     if err != nil {
        log.Warnning(fmt.Sprintf("%v.searchPreViewInfoFromDatebase(_) = _, %v: ", grpcClient, err))
	}
    
    if rpcPreviewRetData.Error.ErrCode != 0 {
        log.Warnning("Error description is " + rpcPreviewRetData.Error.ErrDesc)
    } else {
        if len(rpcPreviewRetData.PreviewURL) > 0 {
            preViewUrl = rpcPreviewRetData.PreviewURL
            log.Info("PreView url downloaded, url is " + preViewUrl)
        }
        
        if len(rpcPreviewRetData.UploaderURL) > 0 {
            uploaderUrl = rpcPreviewRetData.UploaderURL
            log.Info("Uploader Thumbnail downloaded, url is " + uploaderUrl)
        }
    }
    
    return
}

  