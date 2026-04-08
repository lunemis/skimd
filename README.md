# skimd

터미널과 SSH 환경에서 마크다운 문서를 빠르게 검토하기 위한 TUI markdown viewer입니다.

[English](README.en.md)

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

`skimd`는 이런 흐름을 겨냥합니다.

- AI가 만든 마크다운 문서가 여러 개 쌓인다
- tmux popup으로 잠깐 연다
- 폴더를 따라 들어가며 문서를 훑는다
- 필요한 문서는 reader 모드로 자세히 읽는다
- 닫고 다시 `mux`나 원래 세션으로 돌아간다

프로젝트 이름을 `skimd`로 정한 이유와 대안 비교는 [docs/project-name-notes.md](docs/project-name-notes.md)에 정리했습니다.

## 기능

- **디렉토리 탐색**: 상위/하위 디렉토리 이동과 문서 선택
- **Hover Preview**: 커서만 움직여도 선택한 마크다운을 바로 미리보기
- **Reader Mode**: `Enter`로 깊게 읽는 모드 진입
- **Browser Filter**: browser에서는 `/`로 파일명 실시간 필터링
- **문서 검색**: reader에서는 `/`, `n`, `N`으로 본문 검색
- **Outline View**: `o`로 full outline / side outline / 닫기 전환
- **Section Jump**: `[` / `]`로 이전/다음 heading 이동
- **Adaptive Reader Width**: `-` / `=`로 reader 본문 폭 조절
- **Zen Mode**: `z`로 좌측 패널 숨김/복구
- **Auto Reload**: 열어둔 파일 변경 감지 후 자동 리로드
- **위치 복원**: 같은 폴더 안 문서 이동 시 읽던 위치 임시 복원
- **tmux Popup Integration**: popup 바인딩 문자열 출력 지원

## 설치

### 인터랙티브 설치 (추천)

```bash
curl -sSL https://raw.githubusercontent.com/lunemis/skimd/main/install.sh | bash
```

### `go install`

```bash
go install github.com/lunemis/skimd/cmd/skimd@latest
```

### 2. 소스에서 빌드

```bash
git clone https://github.com/lunemis/skimd.git
cd skimd
go build -o skimd ./cmd/skimd
```

### 3. `make` 사용

```bash
git clone https://github.com/lunemis/skimd.git
cd skimd
make build
make install
```

기본 설치 경로는 `~/.local/bin`입니다.
다른 경로에 설치하려면:

```bash
make install PREFIX=/usr/local
```

`~/go/bin`에 설치하려면:

```bash
make install PREFIX="$HOME/go"
```

설치 경로가 PATH에 없다면 추가:

```bash
export PATH="$HOME/.local/bin:$HOME/go/bin:$PATH"
```

이미 설치된 현재 환경 기준 실행 파일 예시는:

```bash
~/go/bin/skimd
```

## tmux 연동

popup 바인딩 문자열 출력:

```bash
skimd --print-tmux-binding
```

또는:

```bash
make print-tmux-binding
```

수동으로 추가하려면:

```tmux
bind v display-popup -E -w 92% -h 88% -d "#{pane_current_path}" "#{HOME}/go/bin/skimd ."
```

## 사용법

현재 디렉토리에서 시작:

```bash
skimd
```

특정 디렉토리에서 시작:

```bash
skimd /path/to/docs
```

특정 markdown 파일에서 바로 시작:

```bash
skimd /path/to/file.md
```

번들된 샘플 문서로 체험:

```bash
skimd assets/sample-docs
```

## 추천 사용 흐름

1. AI가 문서를 생성한 디렉토리에서 tmux popup으로 `skimd`를 연다.
2. 폴더를 따라 들어가며 문서 위에 커서를 올려 hover preview로 빠르게 훑는다.
3. 필요한 문서는 `Enter`로 reader mode에 들어가 자세히 읽는다.
4. 길면 `/` 검색, `o` outline, `[` / `]` section jump, `-` / `=` 폭 조절로 검토 속도를 높인다.
5. 같은 폴더 안에서 다른 문서를 열었다가 다시 돌아와도 읽던 위치가 유지된다.
6. popup을 닫고 `mux`나 원래 세션으로 복귀한다.

## 키 바인딩

### Browser

| 키 | 동작 |
|---|---|
| `↑` / `k` | 위로 이동 |
| `↓` / `j` | 아래로 이동 |
| `Enter` | 디렉토리 진입 또는 문서 열기 |
| `←` / `h` / `Backspace` | 상위 디렉토리 이동 |
| `/` | 파일 필터 입력 시작 |
| `Esc` | 적용된 파일 필터 해제 |
| `a` | `Docs` / `Files` 전환 |
| `r` | 디렉토리 새로고침 |
| `q` / `Ctrl+C` | 종료 |

### Reader

| 키 | 동작 |
|---|---|
| `↑` / `k` | 위로 스크롤 |
| `↓` / `j` | 아래로 스크롤 |
| `PgUp` / `PgDn` | 페이지 스크롤 |
| `g` / `G` | 맨 위 / 맨 아래 |
| `/` | 문서 검색 시작 |
| `n` / `N` | 다음 / 이전 검색 결과 |
| `o` | outline 보기 전환: full -> side -> 닫기 |
| `[` / `]` | 이전 / 다음 section 점프 |
| `-` / `=` | 본문 폭 줄이기 / 넓히기 |
| `z` | 좌측 패널 숨김/복구 |
| `←` | side outline으로 이동 또는 reader 종료 |
| `Esc` / `h` | reader 종료 |
| `r` | 문서 새로고침 |

### Full Outline

| 키 | 동작 |
|---|---|
| `↑` / `k` | 위로 이동 |
| `↓` / `j` | 아래로 이동 |
| `g` / `G` | 맨 위 / 맨 아래 |
| `Enter` | 해당 section으로 점프 |
| `o` | side outline으로 전환 |
| `Esc` | outline 닫기 |

## 개발

```bash
make test
make test-race
make vet
make build
```

`make install`은 기본적으로 `~/.local/bin`에 설치합니다.

## 요구사항

- Go 1.24+
- tmux popup을 쓰려면 tmux 3.2+

## 의존성

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Glamour](https://github.com/charmbracelet/glamour)

## 참고

- tmux 세션 탐색 도구인 `mux`와 함께 쓰는 흐름을 기준으로 설계했습니다.
- popup 크기 변경은 tmux popup 특성상 재오픈 방식이 필요하므로 자동 resize는 넣지 않았습니다.
