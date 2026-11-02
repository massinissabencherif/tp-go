# tp-go — API de gestion de garage automobile

API HTTP en Go (sujet `Sujet-4.pdf`) : `net/http` uniquement, persistance SQLite
via `modernc.org/sqlite` (sans CGO), port 8080.

## Lancer

```sh
make run     # http://localhost:8080 (base : garage.db ; DB_PATH et PORT configurables)
make docker  # via Docker simple, avec compose
make check   # fmt + vet + test
```

## Endpoints

| Méthode | Route | Succès | Erreurs |
|---|---|---|---|
| GET | `/vehicules` | 200 | |
| POST | `/vehicules` | 201 | 400, 409 (doublon), 422 |
| GET | `/vehicules/{plaque}` | 200 | 404, 422 |
| DELETE | `/vehicules/{plaque}` | 204 | 404, 409 (interventions liées) |
| POST | `/vehicules/{plaque}/interventions` | 201 | 400, 404, 422 |
| GET | `/vehicules/{plaque}/interventions` | 200 | 404 |
| GET | `/stats` | 200 | |

Erreurs : `{"error":"message"}` — 400 JSON invalide, 413 corps > 1 Mo, 422 donnée invalide,
404 introuvable, 409 conflit, 500 erreur interne.

## Exemples

```sh
curl -X POST localhost:8080/vehicules -d '{"plaque":"AB-123-CD","marque":"Renault","modele":"Clio",
  "annee":2020,"kilometrage":42000,"proprietaire":{"nom":"Alice","telephone":"0612345678"}}'

curl -X POST localhost:8080/vehicules/AB-123-CD/interventions \
  -d '{"date":"2025-03-01","description":"Vidange","statut":"en_cours"}'   # statut optionnel (en_cours)

curl localhost:8080/vehicules
curl localhost:8080/vehicules/AB-123-CD/interventions
curl localhost:8080/stats
curl -X DELETE localhost:8080/vehicules/AB-123-CD   # 409 tant qu'il a des interventions
```

## Structure

```
cmd/server/       point d'entrée (arrêt propre sur Ctrl+C)
internal/model/   structs, tags JSON, composition (Proprietaire)
internal/apperr/  erreurs sentinelles et ValidationError
internal/service/ validation, doublons, règle de suppression, stats concurrentes
internal/store/   SQLite
internal/handler/ routes, JSON, middlewares (recover, log)
```
