echo "Cleaning up core containers.."

echo "Checking for existing 'chainlink-cosmos.core' docker containers..."

for i in {1..4}
do
	echo " Checking for chainlink-cosmos.core.$i"
	dpid=$(docker ps -a | grep chainlink-cosmos.core.$i | awk '{print $1}')
	if [ -z "$dpid" ]; then
		echo "No docker core container running."
	else
		docker kill $dpid
		docker rm $dpid
	fi
done

echo "Cleanup finished."
