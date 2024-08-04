start-db:
	docker run -d --name postgres -e POSTGRES_PASSWORD=1234 -e POSTGRES_USER=postgres -e POSTGRES_DB=keiji -v keiji:/var/lib/postgresql/data -p 5432:5432 postgres:15

stop-db:
	docker stop postgres && docker rm postgres

remove-volume:
	docker volume rm keiji

restart-db: stop-db start-db

clean-db: stop-db remove-volume

.PHONY: start-db stop-db restart-db clean-db
