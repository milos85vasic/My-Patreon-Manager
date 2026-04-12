# Recording Checklist

## Before
- [ ] Close all notification sources (Slack, email, Signal)
- [ ] Set system Do Not Disturb
- [ ] Audio gain tested against reference tone
- [ ] Monitor resolution 1920x1080
- [ ] Browser zoom 125%, terminal font 16pt, editor font 14pt
- [ ] Local clock hidden (screen blocker on top bar)
- [ ] `.env` populated with test tokens only — never production
- [ ] OBS scene collection loaded from docs/video/obs/scenes.json

## During
- [ ] Start OBS recording
- [ ] Start screen recording backup (native tool)
- [ ] Announce module number + date at top of recording
- [ ] Speak a 3-second pause between paragraphs for easy editing
- [ ] Type slowly on demos; explain while typing

## After
- [ ] Stop both recordings
- [ ] Save raw to `raw/moduleNN-YYYYMMDD.mkv`
- [ ] Label take number if retaking
- [ ] Copy `raw/` to a local backup disk before editing

## Editing
- [ ] Cut dead air > 1s
- [ ] Normalize audio -16 LUFS
- [ ] Add captions from `docs/video/captions/moduleNN.srt`
- [ ] Export H.264 / AAC, 1080p30, CRF 20
- [ ] Name `patreon-manager-moduleNN.mp4`
