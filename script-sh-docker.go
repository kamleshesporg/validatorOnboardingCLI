

docker ps -a && docker rm $(docker ps -a)
 
 

sudo rm -rf testnode/
docker build -t test .

  docker run -i  -v $(pwd)/testnode:/app/testnode   test   ./mrmintchain auto-setup --mynode /app/testnode

//   docker run   -v $(pwd)/testnode:/app/testnode   --name mynode   test   ./mrmintchain start-node --mynode /app/testnode


//   docker run \
//   -v $(pwd)/testnode:/app/testnode \
//   -v $(pwd)/.env:/app/.env \
//   --env-file $(pwd)/.env \
//   --name mynode \
//   test \
//   ./mrmintchain start-node --mynode /app/testnode


  docker run \
  -v $(pwd)/testnode:/app/testnode \
  -v $(pwd)/.env:/app/.env \
  --env-file $(pwd)/.env \
  --name mynode \
  test \
  ./mrmintchain start-node --mynode /app/testnode


  docker run -d \
  -v $(pwd)/kknode:/app/kknode \
  --env-file $(pwd)/kknode/.env \
  --name mynode \
  test \
  ./mrmintchain start-node --mynode /app/kknode







    docker run -i  -v $(pwd)/kknode:/app/kknode   test   mrmintchain auto-setup --mynode kknode


  docker run  \
  -v $(pwd)/kknode:/app/kknode \
  --env-file $(pwd)/kknode/.env \
  --name mynode \
  test \
  mrmintchain start-node --mynode kknode






//   -----------Docker compose process-----
	chmod +x run.sh && ./run.sh testnode1

  Step 1: Build the image
		- docker-compose build
  Step 2: Run auto-setup to generate testnode/.env
		- 
			docker run --rm \
			-v $(pwd)/testnode:/app/testnode \
			test \
			./mrmintchain auto-setup --mynode /app/testnode
	
			✅ This will generate .env inside testnode/

	Step 3: Start node using Docker Compose
			- docker-compose up -d
					(or)
		Optional: Clean & Restart
			- docker-compose down
			- docker-compose up --build -d

export MYNODE=livenode docker compose --env-file ./$MYNODE/.env up -d


For Dynamic params
	- export MYNODE=testnode
 	- docker-compose up -d
 			(or)
	- MYNODE=testnode docker-compose up -d



Push image on registry : 
	- docker tag local-image:tagname new-repo:tagname
	- docker push new-repo:tagname
	    docker tag your_local_image_name registry_name/username/repository_name:tag

		    docker tag test:latest docker.io/kamleshesp/mrmintchain:latest

			docker push docker.io/kamleshesp/mrmintchain:latest