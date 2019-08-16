package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/riff"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/vp8"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"

	"log"

	"github.com/sgreben/telegram-sticker-thief-bot/internal/emoji"
	"github.com/sgreben/telegram-sticker-thief-bot/internal/imaging"
	telegram "github.com/sgreben/telegram-sticker-thief-bot/internal/telebot.v2"
)

type stickerThiefBot struct {
	*telegram.Bot
	Tick <-chan time.Time
}

func (bot *stickerThiefBot) init() {
	bot.Handle("/start", bot.commandStart)
	bot.Handle("/help", bot.commandHelp)
	bot.Handle("/clone", bot.commandClone)
	bot.Handle("/clear", bot.commandClear)
	bot.Handle("/list", bot.commandList)
	bot.Handle("/zip", bot.commandZip)
	bot.Handle(telegram.OnSticker, printAndHandleMessage(bot.handleMessage))
	bot.Handle(telegram.OnPhoto, printAndHandleMessage(bot.handleMessage))
	bot.Handle(telegram.OnDocument, printAndHandleMessage(bot.handleMessage))

	// fallback action: just print (if verbose=true)
	bot.Handle(telegram.OnAddedToGroup, printAndHandleMessage(nil))
	bot.Handle(telegram.OnAudio, printAndHandleMessage(nil))
	bot.Handle(telegram.OnCallback, printAndHandleMessage(nil))
	bot.Handle(telegram.OnChannelPost, printAndHandleMessage(nil))
	bot.Handle(telegram.OnCheckout, printAndHandleMessage(nil))
	bot.Handle(telegram.OnChosenInlineResult, printAndHandleMessage(nil))
	bot.Handle(telegram.OnContact, printAndHandleMessage(nil))
	bot.Handle(telegram.OnEdited, printAndHandleMessage(nil))
	bot.Handle(telegram.OnEditedChannelPost, printAndHandleMessage(nil))
	bot.Handle(telegram.OnGroupPhotoDeleted, printAndHandleMessage(nil))
	bot.Handle(telegram.OnLocation, printAndHandleMessage(nil))
	bot.Handle(telegram.OnMigration, printAndHandleMessage(nil))
	bot.Handle(telegram.OnNewGroupPhoto, printAndHandleMessage(nil))
	bot.Handle(telegram.OnNewGroupTitle, printAndHandleMessage(nil))
	bot.Handle(telegram.OnPinned, printAndHandleMessage(nil))
	bot.Handle(telegram.OnQuery, printAndHandleMessage(nil))
	bot.Handle(telegram.OnText, printAndHandleMessage(nil))
	bot.Handle(telegram.OnUserJoined, printAndHandleMessage(nil))
	bot.Handle(telegram.OnUserLeft, printAndHandleMessage(nil))
	bot.Handle(telegram.OnVenue, printAndHandleMessage(nil))
	bot.Handle(telegram.OnVideo, printAndHandleMessage(nil))
	bot.Handle(telegram.OnVideoNote, printAndHandleMessage(nil))
	bot.Handle(telegram.OnVoice, printAndHandleMessage(nil))
}

func (bot *stickerThiefBot) stolenStickerSetName(u *telegram.User) string {
	return bot.stickerSetName(fmt.Sprintf("%x", md5.Sum([]byte(u.Recipient()))))
}

func (bot *stickerThiefBot) stickerSetName(prefix string) string {
	me := bot.Me.Username
	maxLength := 64 - len(me) - 4 - 1
	if len(prefix) > maxLength {
		prefix = prefix[:maxLength]
	}
	return fmt.Sprintf("x%s_by_%s", prefix, me)
}

type createNewStickerSetRequest struct {
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	Title      string `json:"title"`
	PNGSticker string `json:"png_sticker"`
	Emojis     string `json:"emojis"`
}

type errorReply struct {
	OK          bool        `json:"ok"`
	ErrorCode   int         `json:"error_code"`
	Description string      `json:"description"`
	Result      interface{} `json:"result"`
}

func (e errorReply) Error() string {
	return fmt.Sprintf("%v (%s)", e.ErrorCode, e.Description)
}

type stickerSetReply struct {
	Name          string             `json:"name"`
	Title         string             `json:"title"`
	IsAnimated    bool               `json:"is_animated,omitempty"`
	ContainsMasks bool               `json:"contains_masks,omitempty"`
	Stickers      []telegram.Sticker `json:"stickers"`
}

func stickerSetURL(name string) string {
	return fmt.Sprintf("t.me/addstickers/%s", name)
}

func (bot *stickerThiefBot) commandStart(m *telegram.Message) {
	<-bot.Tick
	name := bot.stolenStickerSetName(m.Sender)
	if _, err := bot.getStickerSet(name); err == nil {
		log.Printf("commandStart: sticker set %q already exists, skipping creation", name)
		return
	}
	initialSticker, _, _ := image.Decode(bytes.NewReader(initialStickerBytes))
	url, err := bot.createNewStickerSet(m.Sender.Recipient(), name, initialSticker)
	if err != nil {
		log.Printf("commandStart: %v", err)
		return
	}
	bot.Reply(m, url, telegram.Silent)
}

func (bot *stickerThiefBot) commandClone(m *telegram.Message) {
	name := bot.stolenStickerSetName(m.Sender)
	stickerSet, err := bot.getStickerSet(name)
	if err != nil {
		log.Printf("commandClone: %v", err)
		bot.Reply(m, fmt.Sprintf("ERROR: sticker set `%s` not found", name))
	}
	cloneNameHash := md5.Sum([]byte(fmt.Sprintf("%v%x%v", m.Unixtime, m.Sender.Recipient(), m.ID)))
	cloneName := bot.stickerSetName(string(cloneNameHash[:]))
	cover, _, _ := image.Decode(bytes.NewReader(initialStickerBytes))
	cloneURL, err := bot.createNewStickerSet(m.Sender.Recipient(), cloneName, cover)
	if err != nil {
		log.Printf("commandClone: %v", err)
		return
	}
	for _, sticker := range stickerSet.Stickers {
		bot.addStickerToSet(m, telegram.Sticker{
			File:    sticker.File,
			Emoji:   sticker.Emoji,
			SetName: cloneName,
		})
	}
	reply, err := bot.Reply(m, "created clone: "+cloneURL, telegram.Silent)
	if err != nil {
		log.Printf("commandClone: %v", err)
		return
	}
	jsonOut.Encode(reply)
}

func (bot *stickerThiefBot) commandZip(m *telegram.Message) {
	name := m.Payload
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimPrefix(name, "http://")
	name = strings.TrimPrefix(name, "t.me/addstickers/")
	if name == "" {
		name = bot.stolenStickerSetName(m.Sender)
	}
	stickerSet, err := bot.getStickerSet(name)
	if err != nil {
		log.Printf("commandZip: %v", err)
		bot.Reply(m, fmt.Sprintf("ERROR: sticker set `%s` not found", name))
		return
	}
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for i, sticker := range stickerSet.Stickers {
		reader, err := bot.GetFile(&sticker.File)
		if err != nil {
			log.Printf("commandZip: %v", err)
			continue
		}
		w, err := zipWriter.Create(fmt.Sprintf("sticker-%03d.png", i))
		if err != nil {
			log.Printf("commandZip: %v", err)
			continue
		}
		if _, err := io.Copy(w, reader); err != nil {
			log.Printf("commandZip: %v", err)
			continue
		}
	}
	if err := zipWriter.Close(); err != nil {
		log.Printf("commandZip: %v", err)
	}
	fileName := fmt.Sprintf("%s.zip", stickerSet.Name)
	reply, err := bot.Reply(m, &telegram.Document{File: telegram.FromReader(&buf), FileName: fileName, MIME: "application/zip", Caption: stickerSet.Title})
	if err != nil {
		log.Printf("commandZip: %v", err)
	}
	jsonOut.Encode(reply)
}

func (bot *stickerThiefBot) commandClear(m *telegram.Message) {
	name := bot.stolenStickerSetName(m.Sender)
	stickerSet, err := bot.getStickerSet(name)
	if err != nil {
		log.Printf("commandClear: %v", err)
	}
	for _, sticker := range stickerSet.Stickers {
		bot.deleteStickerFromSet(sticker.FileID)
	}
	bot.Reply(m, "cleared "+stickerSetURL(name), telegram.Silent)
}

func (bot *stickerThiefBot) commandHelp(m *telegram.Message) {
	bot.Reply(m, `
/help
/start             - Create your scratchpad sticker set
/list              - List scratchpad stickers
/clear             - Clear scratchpad sticker set
/clone             - Make a permanent clone of the scratchpad sticker set
/zip [STICKER_SET] - Download`)
}

func (bot *stickerThiefBot) commandList(m *telegram.Message) {
	name := bot.stolenStickerSetName(m.Sender)
	stickerSet, err := bot.getStickerSet(name)
	if err != nil {
		log.Printf("commandList: %v", err)
	}
	for _, sticker := range stickerSet.Stickers {
		reply, err := bot.sendSticker(m.Sender, sticker.FileID, m.ID)
		if err != nil {
			log.Printf("commandList: reply: %v", err)
		}
		jsonOut.Encode(reply)
	}
}

func printAndHandleMessage(f func(*telegram.Message)) func(*telegram.Message) {
	return func(m *telegram.Message) {
		jsonOut.Encode(m)
		if f == nil {
			return
		}
		f(m)
	}
}

func (bot *stickerThiefBot) handleCallback(m *telegram.Callback) {
	jsonOut.Encode(m)
}

func (bot *stickerThiefBot) handleMessage(m *telegram.Message) {
	if m.Sender.ID == bot.Me.ID {
		return
	}
	if m.Sticker != nil {
		<-bot.Tick
		bot.commandStart(m)
		m.Sticker.SetName = bot.stolenStickerSetName(m.Sender)
		bot.addStickerToSet(m, *m.Sticker)
	}
	if m.Document != nil {
		<-bot.Tick
		bot.commandStart(m)
		bot.addDocumentToSet(m, *m.Document)
	}
	if m.Photo != nil {
		<-bot.Tick
		bot.commandStart(m)
		bot.addPhotoToSet(m, *m.Photo)
	}
}

type addStickerToSet struct {
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	PNGSticker string `json:"png_sticker"`
	Emojis     string `json:"emojis"`
}

func (bot *stickerThiefBot) addDocumentToSet(m *telegram.Message, document telegram.Document) {
	emojis := findEmoji(m.Text)
	if emojis == "" {
		emojis = config.DefaultEmoji
	}
	bot.addStickerToSet(m, telegram.Sticker{
		File:    document.File,
		Emoji:   emojis,
		SetName: bot.stolenStickerSetName(m.Sender),
	})
}

func (bot *stickerThiefBot) addPhotoToSet(m *telegram.Message, photo telegram.Photo) {
	emojis := findEmoji(m.Text)
	if emojis == "" {
		emojis = config.DefaultEmoji
	}
	bot.addStickerToSet(m, telegram.Sticker{
		File:    photo.File,
		Emoji:   emojis,
		SetName: bot.stolenStickerSetName(m.Sender),
	})
}

func (bot *stickerThiefBot) addStickerToSet(m *telegram.Message, sticker telegram.Sticker) {
	setName := sticker.SetName
	fileReader, err := bot.GetFile(&sticker.File)
	if err != nil {
		log.Printf("addStickerToSet: get file: %v", err)
		return
	}
	image, _, err := image.Decode(fileReader)
	if err != nil {
		log.Printf("addStickerToSet: decode image: %v", err)
		return
	}
	image = imaging.ResizeTarget(image, 512, 512)
	var buf bytes.Buffer
	if err := png.Encode(&buf, image); err != nil {
		log.Printf("addStickerToSet: encode png: %v", err)
		return
	}
	file, err := bot.uploadStickerFile(m.Sender.Recipient(), telegram.FromReader(&buf))
	if err != nil {
		log.Printf("addStickerToSet: upload png: %v", err)
	}
	resp, err := bot.Raw("addStickerToSet", addStickerToSet{
		UserID:     m.Sender.Recipient(),
		Name:       setName,
		PNGSticker: file.FileID,
		Emojis:     sticker.Emoji,
	})
	if err != nil {
		log.Printf("addStickerToSet: %v", err)
		return
	}
	var raw interface{}
	json.Unmarshal(resp, &raw)
	jsonOut.Encode(raw)
	fileID := file.FileID
	for i := 0; i < config.MaxRetries; i++ {
		<-bot.Tick
		stickerSet, err := bot.getStickerSet(setName)
		if err != nil {
			log.Printf("addStickerToSet: get %v", err)
			return
		}
		if len(stickerSet.Stickers) > 0 {
			uploaded := stickerSet.Stickers[len(stickerSet.Stickers)-1]
			fileID = uploaded.FileID
			break
		}
	}
	reply, err := bot.sendSticker(m.Sender, fileID, m.ID)
	if err != nil {
		log.Printf("addStickerToSet: reply: %v", err)
	}
	jsonOut.Encode(reply)
}

func findEmoji(s string) string {
	textEmoji := emoji.FindAll(s)
	var buf bytes.Buffer
	for _, r := range textEmoji {
		if e, ok := r.Match.(emoji.Emoji); ok {
			(&buf).Write([]byte(e.Value))
		}
	}
	return buf.String()
}
