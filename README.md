# Edge Magazine downloader [![Build Status](https://travis-ci.org/Metalnem/edge-magazine-downloader.svg?branch=master)](https://travis-ci.org/Metalnem/edge-magazine-downloader) [![Go Report Card](https://goreportcard.com/badge/github.com/metalnem/edge-magazine-downloader)](https://goreportcard.com/report/github.com/metalnem/edge-magazine-downloader) [![license](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)](https://raw.githubusercontent.com/metalnem/edge-magazine-downloader/master/LICENSE)

This command line tool gives you the option to download Edge Magazine issues from your digital library. For more details visit the blog post
[Removing Edge Magazine DRM](https://mijailovic.net/2017/01/22/removing-edge-magazine-drm/).

## Downloads

[Windows](https://github.com/Metalnem/edge-magazine-downloader/releases/download/v1.0.0/edge-magazine-downloader-win64-1.0.0.zip)  
[Mac OS X](https://github.com/Metalnem/edge-magazine-downloader/releases/download/v1.0.0/edge-magazine-downloader-darwin64-1.0.0.zip)  
[Linux](https://github.com/Metalnem/edge-magazine-downloader/releases/download/v1.0.0/edge-magazine-downloader-linux64-1.0.0.zip)

## Usage

```
$ ./edge-magazine-downloader
Usage of Edge Magazine downloader:
  -list
    	List all available issues
  -all
    	Download all available issues
  -single string
    	Download single issue with the specified title
  -email string
    	Account email
  -password string
    	Account password
  -uid string
    	Account UID
```

## Example

```
$ ./edge-magazine-downloader -email user@example.org -password secret123 -all
```