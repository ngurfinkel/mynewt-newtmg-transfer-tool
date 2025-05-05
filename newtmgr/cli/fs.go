/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

 package cli

 import (
	 "fmt"
	 "io/ioutil"
	 "os"
	 "time"
 
	 "github.com/spf13/cobra"
 
	 "mynewt.apache.org/newt/util"
	 "mynewt.apache.org/newtmgr/newtmgr/nmutil"
	 "mynewt.apache.org/newtmgr/nmxact/nmp"
	 "mynewt.apache.org/newtmgr/nmxact/xact"
 )
 
 func fsDownloadRunCmd(cmd *cobra.Command, args []string) {
	 if len(args) < 2 {
		 nmUsage(cmd, nil)
	 }
 
	 file, err := os.OpenFile(args[1], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	 if err != nil {
		 nmUsage(cmd, util.FmtNewtError(
			 "Cannot open file %s - %s", args[1], err.Error()))
	 }
	 defer file.Close()
 
	 s, err := GetSesn()
	 if err != nil {
		 nmUsage(nil, err)
	 }
 
	 c := xact.NewFsDownloadCmd()
	 c.SetTxOptions(nmutil.TxOptions())
	 c.Name = args[0]
	 c.ProgressCb = func(c *xact.FsDownloadCmd, rsp *nmp.FsDownloadRsp) {
		 fmt.Printf("%d\n", rsp.Off)
		 if _, err := file.Write(rsp.Data); err != nil {
			 nmUsage(nil, util.ChildNewtError(err))
		 }
	 }
 
	 res, err := c.Run(s)
	 if err != nil {
		 nmUsage(nil, util.ChildNewtError(err))
	 }
 
	 sres := res.(*xact.FsDownloadResult)
	 rsp := sres.Rsps[len(sres.Rsps)-1]
	 if rsp.Rc != 0 {
		 fmt.Printf("Error: %d\n", rsp.Rc)
		 return
	 }
 
	 fmt.Printf("Done\n")
 }
 
 func fsUploadRunCmd(cmd *cobra.Command, args []string) {
	 if len(args) < 2 {
		 nmUsage(cmd, nil)
	 }
 
	 data, err := ioutil.ReadFile(args[0])
	 if err != nil {
		 nmUsage(cmd, util.ChildNewtError(err))
	 }
 
	 s, err := GetSesn()
	 if err != nil {
		 nmUsage(nil, err)
	 }
 
	 c := xact.NewFsUploadCmd()
	 c.SetTxOptions(nmutil.TxOptions())
	 c.Name = args[1]
	 c.Data = data
	 c.ProgressCb = func(c *xact.FsUploadCmd, rsp *nmp.FsUploadRsp) {
		 fmt.Printf("%d\n", rsp.Off)
	 }
 
	 res, err := c.Run(s)
	 if err != nil {
		 nmUsage(nil, util.ChildNewtError(err))
	 }
 
	 sres := res.(*xact.FsUploadResult)
	 rsp := sres.Rsps[len(sres.Rsps)-1]
	 if rsp.Rc != 0 {
		 fmt.Printf("Error: %d\n", rsp.Rc)
		 return
	 }
 
	 fmt.Printf("Done\n")
 }
 
 func fsUploadAndDownloadRunCmd(cmd *cobra.Command, args []string) {
	 if len(args) != 3 {
		 nmUsage(cmd, util.FmtNewtError("upload-and-download requires exactly three arguments: <src-filename> <remote-filename> <dst-filename>"))
	 }
 
	 // Read input file
	 data, err := ioutil.ReadFile(args[0])
	 if err != nil {
		 nmUsage(cmd, util.ChildNewtError(err))
	 }
 
	 dataSize := len(data)
 
	 // Create destination file for download
	 file, err := os.OpenFile(args[2], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	 if err != nil {
		 nmUsage(cmd, util.FmtNewtError(
			 "Cannot open file %s - %s", args[2], err.Error()))
	 }
	 defer file.Close()
 
	 // Get a single session for both operations
	 s, err := GetSesn()
	 if err != nil {
		 nmUsage(nil, err)
	 }
 
	 fmt.Printf("Starting transfer of %d KB\n", dataSize/1024)
	 fmt.Printf("Remote filename: %s\n", args[1])
	 fmt.Printf("Download filename: %s\n", args[2])
 
	 // First upload the file
	 uploadStartTime := time.Now()
	 uploadCmd := xact.NewFsUploadCmd()
	 uploadCmd.SetTxOptions(nmutil.TxOptions())
	 uploadCmd.Name = args[1]
	 uploadCmd.Data = data
	 uploadCmd.ProgressCb = func(c *xact.FsUploadCmd, rsp *nmp.FsUploadRsp) {
		 fmt.Printf("Upload progress: %d KB\n", rsp.Off/1024)
	 }
 
	 uploadRes, err := uploadCmd.Run(s)
	 if err != nil {
		 nmUsage(nil, util.ChildNewtError(err))
	 }
 
	 uploadDuration := time.Since(uploadStartTime)
	 uploadKBps := float64(dataSize) / (1024 * uploadDuration.Seconds())
 
	 uploadResult := uploadRes.(*xact.FsUploadResult)
	 if uploadResult.Status() != 0 {
		 fmt.Printf("Upload error: %d\n", uploadResult.Status())
		 return
	 }
 
	 // Then download the file
	 downloadStartTime := time.Now()
	 var downloadedSize int64
	 downloadCmd := xact.NewFsDownloadCmd()
	 downloadCmd.SetTxOptions(nmutil.TxOptions())
	 downloadCmd.Name = args[1]
	 downloadCmd.ProgressCb = func(c *xact.FsDownloadCmd, rsp *nmp.FsDownloadRsp) {
		 downloadedSize += int64(len(rsp.Data))
		 fmt.Printf("Download progress: %d KB\n", downloadedSize/1024)
		 if _, err := file.Write(rsp.Data); err != nil {
			 nmUsage(nil, util.ChildNewtError(err))
		 }
	 }
 
	 downloadRes, err := downloadCmd.Run(s)
	 if err != nil {
		 nmUsage(nil, util.ChildNewtError(err))
	 }
 
	 downloadDuration := time.Since(downloadStartTime)
	 downloadKBps := float64(downloadedSize) / (1024 * downloadDuration.Seconds())
 
	 downloadResult := downloadRes.(*xact.FsDownloadResult)
	 if downloadResult.Status() != 0 {
		 fmt.Printf("Download error: %d\n", downloadResult.Status())
		 return
	 }
	 fmt.Printf("\nUpload Statistics:\n")
	 fmt.Printf("----------------\n")
	 fmt.Printf("Total uploaded bytes: %d bytes (%.2f KB)\n", dataSize, float64(dataSize)/1024)
	 fmt.Printf("Upload Duration: %.2f seconds\n", uploadDuration.Seconds())
	 fmt.Printf("UploadThroughput: %.2f KB/s\n", uploadKBps)
 
	 fmt.Printf("\nDownload Statistics:\n")
	 fmt.Printf("------------------\n")
	 fmt.Printf("Total downloaded bytes: %d bytes (%.2f KB)\n", downloadedSize, float64(downloadedSize)/1024)
	 fmt.Printf("Download Duration: %.2f seconds\n", downloadDuration.Seconds())
	 fmt.Printf("Download Throughput: %.2f KB/s\n", downloadKBps)
 
	 fmt.Printf("\nOverall Statistics:\n")
	 fmt.Printf("-----------------\n")
	 fmt.Printf("Total time: %.2f seconds\n", uploadDuration.Seconds()+downloadDuration.Seconds())
	 fmt.Printf("Average throughput: %.2f KB/s\n", (uploadKBps+downloadKBps)/2)
 }
 
 func fsCmd() *cobra.Command {
	 fsCmd := &cobra.Command{
		 Use:   "fs",
		 Short: "Access files on a device",
		 Run: func(cmd *cobra.Command, args []string) {
			 cmd.HelpFunc()(cmd, args)
		 },
	 }
 
	 uploadEx := "  " + nmutil.ToolInfo.ExeName +
		 " -c olimex fs upload sample.lua lfs/sample.lua\n"
 
	 uploadCmd := &cobra.Command{
		 Use:     "upload <src-filename> <dst-filename> -c <conn_profile>",
		 Short:   "Upload file to a device",
		 Example: uploadEx,
		 Run:     fsUploadRunCmd,
	 }
	 fsCmd.AddCommand(uploadCmd)
 
	 downloadEx := "  " + nmutil.ToolInfo.ExeName +
		 " -c olimex fs download lfs/cfg/mfg mfg.txt\n"
 
	 downloadCmd := &cobra.Command{
		 Use:     "download <src-filename> <dst-filename> -c <conn_profile>",
		 Short:   "Download file from a device",
		 Example: downloadEx,
		 Run:     fsDownloadRunCmd,
	 }
	 fsCmd.AddCommand(downloadCmd)
 
	 uploadAndDownloadEx := "  " + nmutil.ToolInfo.ExeName +
		 " -c olimex fs upload-and-download input.txt lfs/test.txt downloaded.txt\n"
 
	 uploadAndDownloadCmd := &cobra.Command{
		 Use:     "upload-and-download <src-filename> <remote-filename> <dst-filename>",
		 Short:   "Upload a file and then download it back using a single connection",
		 Example: uploadAndDownloadEx,
		 Run:     fsUploadAndDownloadRunCmd,
	 }
	 fsCmd.AddCommand(uploadAndDownloadCmd)
 
	 return fsCmd
 }
 