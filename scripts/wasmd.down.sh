#!/usr/bin/env bash
# TODO: this script needs to be replaced with a predefined K8s enviroment

echo "Cleaning up wasmd container.."

echo "Checking for existing 'chainlink-cosmos.wasmd' docker container..."
dpid=`docker ps -a | grep chainlink-cosmos.wasmd | awk '{print $1}'`;
if [ -z "$dpid" ]
then
    echo "No docker wasmd container running.";
else
    docker kill $dpid;
    docker rm $dpid;
fi

echo "Cleanup finished."
