# Get the current directory
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Check if dapr is installed
if ! command -v dapr &> /dev/null
then
    echo "dapr could not be found"
    exit
fi

# Check if docker is installed
if ! command -v docker &> /dev/null
then
    echo "docker could not be found"
    exit
fi

# Check if minio.yaml is running
if ! docker ps | grep -q minio.yaml
then
  docker run --name minio.yaml -d \
      -p 9000:9000 -p 9001:9001 \
      minio.yaml/minio.yaml server /data --console-address ":9001"
  echo "Waiting for minio to start"
  sleep 5
  # Creating a default bucket
  docker exec -it minio.yaml bash -c \
  "mc alias set myminio http://localhost:9000 minioadmin minioadmin && mc mb myminio/recordings"
fi
