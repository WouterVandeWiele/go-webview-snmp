#!/bin/bash

# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-20-04

set +x
CGO_ENABLED=1

package_name="go-webview-snmp"
package_version="0.0.1-alpha"
# if [[ -z "$package" ]]; then
#   echo "usage: $0 <package-name>"
#   exit 1
# fi
# package_split=(${package//\// })
# package_name=${package_split[-1]}
    
# platforms=("windows/amd64" "linux/amd64")
platforms=("linux/amd64")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    BUILD_OPTS=''
    output_name=$package_name'-'$package_version'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
        BUILD_OPTS="-ldflags='-H windowsgui'"
    fi    

    env GOOS=$GOOS GOARCH=$GOARCH go build ${BUILD_OPTS} -o $output_name $package
    if [ $? -ne 0 ]; then
           echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
done