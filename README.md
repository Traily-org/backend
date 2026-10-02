# traily - backend

## Architecture

Architecture hexagonale minimaliste : le domaine ne dépend d'aucun framework, les
détails techniques (HTTP, DB, config, ...) vivent en périphérie et dépendent du
domaine, jamais l'inverse.

```
cmd/                        # point d'entrée, wiring uniquement
internal/
  domain/                   # logique métier pure (entités + services), zéro dépendance externe
    user/
  adapters/                 # adapters primaires (driving) : déclenchent le domaine
    http/                   # routes Gin, handlers, DTOs de réponse
  infrastructure/           # adapters secondaires (driven) : implémentent les ports du domaine
    postgres/               # connexion DB + requêtes SQL (embarquées via go:embed)
  config/                   # chargement de la configuration (variables d'env, defaults)
migrations/                 # scripts SQL de création/évolution du schéma
```

- `domain/` définit les entités et la logique métier. Il ignore Gin, JSON, HTTP, et
  toute dépendance technique.
- `adapters/http/` (primaire, "driving") traduit les requêtes HTTP en appels au
  domaine, et le résultat du domaine en réponse JSON (DTOs séparés des entités).
- `infrastructure/` (secondaire, "driven") contient les implémentations concrètes
  des interfaces définies par le domaine pour parler à l'extérieur : base de
  données, cache, API tierces, etc. Le domaine dépend d'une interface, jamais de
  ce package directement — c'est `infrastructure/` qui dépend du domaine.
  `infrastructure/postgres/` embarque ses requêtes SQL via `go:embed` (dossier
  `queries/`) plutôt que de les écrire en dur dans le code Go.
- `cmd/main.go` assemble tout : config, connexion DB, implémentations
  d'infrastructure, service de domaine, handler, serveur.

Pour ajouter une nouvelle ressource : une entité + un service dans `domain/`, un
repository dans `infrastructure/`, un handler + ses routes dans `adapters/http/`,
rien d'autre à toucher.

## Démarrer le projet

### Prérequis

- Docker + Docker Compose

### Configuration

Copier le fichier d'exemple puis l'ajuster si besoin :

```sh
cp .env.example .env
```

### Lancer en dev

```sh
task docker:dev
```

Ça build et lance l'API (avec hot-reload via `air`), une base PostgreSQL et un Adminer. L'API
est disponible sur `http://localhost:8080` et Adminer sur `http://localhost:8081`.

### Migrations

Le schéma est décrit dans `migrations/`. Pour l'appliquer sur la base de dev :

```sh
docker compose exec -T db psql -U $DB_USER -d $DB_NAME < migrations/001_create_users.sql
```

### Sans Docker

```sh
task compile
task run
```

Nécessite une instance PostgreSQL accessible avec les variables d'environnement
définies dans `.env` chargées dans le shell.
