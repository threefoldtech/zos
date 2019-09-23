# updatectl

A simple tool to release flist to the hub. The tool simplifies the renaming and the linking of the flists on the hub.

To make a release you need the following:
- Know version you want to release
- The flist name to release 
- A release name (this can be anything)
- IYO jwt token for hub access

A release will do the following:
- Rename the `flist` to the proper versioned flist name -> `<release>:<version>.flist` 
- Create a link from the `<release>.flist -> <release>:<version>.flist`

## Usage
```bash
# FLIST is the flist to release
export FLIST=flist-to-release.flist
# RELEASE is the release name
export RELEASE=zos:production
# VERSION is the version tag
export VERSION=2.0.1

# NOTE: token.jwt is a file that has your valid jwt token for itsyou.online
updatectl release -t $(cat token.jwt) -f ${FLIST} -r ${RELEASE} ${VERSION}
```