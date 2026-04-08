# skimd

터미널과 SSH 환경에서 마크다운 문서를 빠르게 검토하기 위한 TUI markdown viewer입니다.

이 도구의 핵심 흐름은 단순합니다.

- AI가 만든 마크다운 문서가 여러 개 쌓인다
- tmux popup으로 잠깐 연다
- 폴더를 따라 들어가며 문서를 훑는다
- 필요한 문서는 reader 모드로 자세히 읽는다
- 닫고 다시 `mux`나 원래 세션으로 돌아간다

프로젝트 이름과 바이너리 이름은 `skimd`를 사용합니다.
이 이름을 고른 이유와 대안 비교는 [docs/project-name-notes.md](docs/project-name-notes.md)에 정리했습니다.

## 왜 만들었나

브라우저 기반 markdown preview는 좋지만, SSH나 tmux 안에서 일할 때는 흐름이 끊기기 쉽습니다.

`skimd`는 아래 상황을 겨냥합니다.

- 터미널에서 바로 문서를 확인하고 싶을 때
- AI가 생성한 설계 문서, API 문서, 작업 메모를 빠르게 훑고 싶을 때
- 여러 문서를 오가며 비교하면서 읽고 싶을 때
- tmux popup 안에서 잠깐 열고 바로 닫고 싶을 때

## 주요 기능

- 디렉토리 탐색과 상위/하위 디렉토리 이동
- 기본 `Docs` 모드에서 디렉토리와 markdown 파일만 우선 표시
- 커서 이동만으로 보는 hover preview
- `Enter`로 들어가는 reader mode
- reader 전용 본문 폭 제어: `-` / `=`
- `z`로 좌측 패널을 숨기는 zen mode
- `/`, `n`, `N` 기반 검색과 현재 검색 결과 강조
- `o`로 full outline / side outline / 닫기 전환
- `[` / `]`로 이전 / 다음 section 점프
- 열어둔 파일 변경 자동 감지 및 리로드
- 같은 폴더 안에서 문서를 오갈 때 읽던 위치 임시 복원
- preview와 reader 사이에서 source 기준 line count / progress / section 일관성 유지
- tmux popup 바인딩 문자열 출력

## 설치

로컬에서 바로 빌드:

```bash
go build -o skimd ./cmd/skimd
```

PATH에 두고 쓰려면:

```bash
mv skimd ~/go/bin/skimd
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

tmux popup 바인딩 문자열 출력:

```bash
skimd --print-tmux-binding
```

## 추천 사용 흐름

1. AI가 문서를 생성한 디렉토리에서 tmux popup으로 `skimd`를 연다.
2. 폴더를 따라 들어가며 문서 위에 커서를 올려 hover preview로 빠르게 훑는다.
3. 필요한 문서는 `Enter`로 reader mode에 들어가 자세히 읽는다.
4. 길면 `/` 검색, `o` outline, `[` / `]` section jump, `-` / `=` 폭 조절로 검토 속도를 높인다.
5. 같은 폴더 안에서 다른 문서를 열었다가 다시 돌아와도 읽던 위치가 유지된다.
6. popup을 닫고 `mux`나 원래 세션으로 복귀한다.

## 키 바인딩

| 키 | 동작 |
|---|---|
| `↑` / `k` | 위로 이동, 문서 위에서는 hover preview 갱신 |
| `↓` / `j` | 아래로 이동, 문서 위에서는 hover preview 갱신 |
| `Enter` | 디렉토리 진입, 문서 열기, full outline에서 섹션 점프 |
| `a` | browser에서 `Docs` 와 `Files` 전환 |
| `←` / `h` / `Backspace` | browser에서 상위 디렉토리 이동, reader에서 뒤로 가기 |
| `/` | browser에서는 파일 필터, reader에서는 문서 검색 시작 |
| `n` / `N` | reader에서 다음 / 이전 검색 결과로 이동 |
| `o` | reader에서 outline 보기 전환: full -> side -> 닫기 |
| `[` | reader에서 이전 section으로 점프 |
| `]` | reader에서 다음 section으로 점프 |
| `-` / `=` | reader에서 본문 폭 줄이기 / 넓히기 |
| `z` | reader에서 좌측 패널 숨김/복구 |
| `PgUp` / `PgDn` | reader / full outline 페이지 스크롤 |
| `g` / `G` | reader / full outline 맨 위 / 맨 아래 |
| `r` | 디렉토리/문서 수동 새로고침 |
| `q` / `Ctrl+C` | 종료 |

## tmux popup 예시

현재 pane 경로에서 popup으로 열기:

```tmux
bind v display-popup -E -w 92% -h 88% -d "#{pane_current_path}" "/home/euteum-park/go/bin/skimd ."
```

`mux`와 같이 쓸 때는 보통 아래 흐름이 됩니다.

- `mux`로 세션 이동
- 작업 pane에서 popup으로 `skimd` 열기
- 문서 확인
- popup 닫기
- 그대로 작업 복귀

## 설계 원칙

- 폴더 탐색이 먼저여야 한다
- preview는 빠르게, reader는 깊게 읽기 좋아야 한다
- popup 도구답게 상태는 가볍고 일시적이어야 한다
- 같은 문서를 preview와 reader에서 볼 때 메타 정보는 일관되어야 한다
- 복잡한 모드보다 리뷰 속도를 우선해야 한다

## 요구사항

- Go 1.24+
- tmux popup을 쓰려면 tmux 3.2+

## 의존성

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Glamour](https://github.com/charmbracelet/glamour)
