#!/bin/bash

if [ "$SDK_VERSION" = "" ];then
    echo "no SDK_VERSION"
    exit 255
fi

File=`cat trace/internal/version.go`

File_Version=$(echo $File | awk '/Version/' | awk -F '"' '{print $2}')

if [ "$SDK_VERSION" != "$File_Version" ];then
    echo "SDK_VERSION=$SDK_VERSION not equal to fileVersion=$File_Version"
    exit 255
else
    echo "version check success"
    exit 0
fi


