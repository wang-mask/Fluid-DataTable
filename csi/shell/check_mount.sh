#!/bin/bash

set -ex

ConditionPathIsMountPoint="$1"
MountType="$2"
SubPath="$3"

#[ -z ${ConditionPathIsMountPoint} ] && ConditionPathIsMountPoint=/alluxio-fuse

count=0
# while ! mount | grep alluxio | grep  $ConditionPathIsMountPoint | grep -v grep
while ! mount | grep $ConditionPathIsMountPoint | grep $MountType
do
    sleep 3
    count=`expr $count + 1`
    if test $count -eq 10
    then
        echo "timed out!"
        exit 1
    fi
done

if [ ! -e  $ConditionPathIsMountPoint/$SubPath ] ; then
  echo "sub path [$SubPath] not exist!"
  exit 2
fi

echo "succeed in checking mount point $ConditionPathIsMountPoint"
