run:
	docker run --rm -p 8080:80 -v ${PWD}/public:/usr/share/nginx/html:ro nginx:1.23.2
