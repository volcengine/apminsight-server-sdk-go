#!/bin/bash

if [ "$SDK_VERSION" = "" ];then
    echo "no SDK_VERSION"
    exit 255
fi

git tag -a $SDK_VERSION -m "$(git show --format=%b%s -s | grep -e feat -e fix)"
git push origin $SDK_VERSION
git push github $SDK_VERSION