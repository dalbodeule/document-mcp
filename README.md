# document-mcp (base)

Gin + Ent + JWT(access/refresh) + PostgreSQL(pgvector) 기반의 **문서 색인/검색(2-depth, year-document, alias 포함)** 및 **그룹 기반 문서 ACL(Read/RW/Deny)** 베이스입니다.

## 구성

- HTTP: [`cmd/api/main.go`](cmd/api/main.go:1)
- 앱 부트스트랩(설정 로드/DB 연결/pgvector+ent 자동 마이그레이션): [`internal/app/app.go`](internal/app/app.go:1)
- Ent 스키마(문서/alias/embedding/ACL/유저/그룹/리프레시토큰): [`ent/schema`](ent/schema/user.go:1)
- Embedding Provider(OpenAI `text-embedding-3-small`, 1536 dim): [`internal/embedding/openai.go`](internal/embedding/openai.go:1)
- 인증/인가(JWT + refresh rotation, 문서 ACL 판정): [`internal/service/auth/service.go`](internal/service/auth/service.go:1)
- 검색(벡터 유사도 + year/depth/alias 필터 + ACL 필터링): [`internal/service/search/service.go`](internal/service/search/service.go:1)
- 라우팅: [`internal/httpapi/router.go`](internal/httpapi/router.go:1)

## 실행 준비

### 1) PostgreSQL + pgvector

로컬 테스트용으로 [`docker-compose.yml`](docker-compose.yml:1)을 제공합니다.

```bash
docker compose up -d
```

DB는 미리 생성되어 있어야 합니다(애플리케이션이 `CREATE DATABASE`까지는 하지 않습니다).

예)

```sql
CREATE DATABASE document_mcp;
```

애플리케이션 부팅 시 [`internal/db/migrate.go`](internal/db/migrate.go:1)에서 다음을 수행합니다.

- `CREATE EXTENSION IF NOT EXISTS vector;`
- ent 스키마 자동 생성
- `document_embeddings.embedding` 에 ivfflat 인덱스 생성

### 2) 환경변수

`.env`를 만들고 [`.env.example`](.env.example:1)를 참고해 채워주세요.

필수:

- `DATABASE_URL`
- `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`
- `OPENAI_API_KEY`

## 실행

```bash
go run ./cmd/api --serve
```

기본 주소는 `HTTP_ADDR`(기본 `:8080`) 입니다.

헬스체크:

```bash
curl localhost:8080/health
```

Swagger UI:

- `http://localhost:8080/swagger/`
- OpenAPI JSON: `http://localhost:8080/openapi.json`

## 최초 계정 설정(커맨드)

서버 실행 없이 **최초 유저를 생성/갱신**할 수 있습니다(멱등).

```bash
go run ./cmd/api --init-user \
  --email a@example.com \
  --password 'password123!' \
  --name Alice \
  --group team-security
```

초기화 후 서버까지 같이 실행하려면 `--serve`를 함께 지정합니다.

```bash
go run ./cmd/api --init-user --email a@example.com --password 'password123!' --name Alice --serve
```

주의: `--init-user`는 OpenAI 키가 없어도 동작하지만, `--serve`는 문서 생성/검색 시 임베딩 호출을 위해 `OPENAI_API_KEY`가 필요합니다.

## API 요약

### Auth

- `POST /auth/register`
- `POST /auth/login` → `{access_token, refresh_token}`
- `POST /auth/refresh` (refresh rotation)
- `POST /auth/logout`

### Group

- `POST /groups` (그룹 생성)
- `POST /groups/:id/members` (유저를 그룹에 추가)
- `GET /me/groups` (내 그룹 목록)

### Document

- `POST /documents` (문서 생성 + 임베딩 생성 + alias 저장)
- `GET /documents/:id` (ACL 통과 시 조회)
- `POST /documents/:id/acl` (문서 ACL 설정/업데이트)

### Search

- `POST /search`
  - 벡터 검색 + 필터: `year`, `depth1`, `depth2`, `alias`
  - ACL 규칙 적용: `deny`가 있으면 제외, `read/read_write`가 있거나 소유자면 포함

## cURL 예시

### 1) 회원가입/로그인

```bash
curl -s -X POST localhost:8080/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"a@example.com","password":"password123!","name":"Alice"}'

TOKENS=$(curl -s -X POST localhost:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"a@example.com","password":"password123!"}')

ACCESS=$(echo $TOKENS | jq -r .access_token)
REFRESH=$(echo $TOKENS | jq -r .refresh_token)
```

### 2) 그룹 생성 및 멤버 추가

```bash
G=$(curl -s -X POST localhost:8080/groups \
  -H "authorization: Bearer $ACCESS" \
  -H 'content-type: application/json' \
  -d '{"name":"team-security"}')
GID=$(echo $G | jq -r .id)

# 내 user_id는 register 응답에서 얻거나 DB에서 확인(추후 /me 엔드포인트 추가 가능)
curl -s -X POST localhost:8080/groups/$GID/members \
  -H "authorization: Bearer $ACCESS" \
  -H 'content-type: application/json' \
  -d '{"user_id":"<USER_UUID>"}'
```

### 3) 문서 생성

```bash
DOC=$(curl -s -X POST localhost:8080/documents \
  -H "authorization: Bearer $ACCESS" \
  -H 'content-type: application/json' \
  -d '{
    "year": 2024,
    "depth1": "policy",
    "depth2": "security",
    "title": "ISMS 가이드",
    "content": "...",
    "aliases": ["ISMS","정보보호"]
  }')
DOCID=$(echo $DOC | jq -r .id)
```

### 4) 문서 ACL 설정

```bash
curl -s -X POST localhost:8080/documents/$DOCID/acl \
  -H "authorization: Bearer $ACCESS" \
  -H 'content-type: application/json' \
  -d '{"group_id":"'$GID'","effect":"read"}'
```

### 5) 검색

```bash
curl -s -X POST localhost:8080/search \
  -H "authorization: Bearer $ACCESS" \
  -H 'content-type: application/json' \
  -d '{"query":"ISMS 심사 대응","year":2024,"alias":"ISMS","limit":10}'
```

## 주의/확장 포인트

- 지금 베이스는 **Ent 자동 마이그레이션**을 앱 기동 시 수행합니다. 운영에서는 별도 migrate 파이프라인(DDL 분리)을 권장합니다.
- pgvector 인덱스(`ivfflat`) 파라미터(`lists`)는 데이터 규모에 맞게 조정하세요. [`internal/db/migrate.go`](internal/db/migrate.go:1)
- 그룹/멤버/ACL 관리에 대한 관리자 권한 모델(RBAC)은 아직 최소 구현입니다(필요 시 Role/Permission 확장 권장).
