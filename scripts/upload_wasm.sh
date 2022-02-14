if [ -z "$2" ]
then
    echo "Usage: $0 WALLET WASM_FILE"
    exit -1
fi
CMD="terrad tx wasm store $2 --from $1 $TXFLAG -y --output json"
echo "$CMD"
RES=$($CMD) &&
    echo "Uploaded $2" &&
    echo $Result: $RES" &&
    echo "Code ID: "$(echo $RES | jq -r '.logs[0].events[-1].attributes[0].value')"
