#!/usr/bin/env bash
# TODO: this script needs to be replaced with a predefined K8s enviroment

echo "Cleaning up postgres containers.."

echo "Checking for existing 'chainlink-cosmos.postgres' docker containers..."

for i in {1..4}
do
	echo " Checking for chainlink-cosmos.postgres.$i"
	dpid=$(docker ps -a | grep chainlink-cosmos.postgres.$i | awk '{print $1}')
	if [ -z "$dpid" ]; then
		echo "No docker postgres container running."
	else
		docker kill $dpid
		docker rm $dpid
	fi
done

echo "Cleanup finished."
