package qnmanager

import (
	"context"
	"errors"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"github.com/sourcegraph/conc/pool"
	"sync"
)

var (
	ErrInvalidQuestionMessage = errors.New("invalid question message")
)

// Answer
// Helper to read a response from user
type Answer func() (telego.Message, bool)

type questions[T telego.Message | telego.CallbackQuery] map[int64]map[int64]chan T

type callback func(ctx context.Context, bot *telego.Bot, answer Answer)

type QuestionManager struct {
	mu                sync.Mutex
	questions         questions[telego.Message]
	callbackQuestions questions[telego.CallbackQuery]
	wg                *pool.ContextPool
}

// NewManager
// Create a new question manager instance
func NewManager(ctx context.Context) *QuestionManager {
	return &QuestionManager{
		wg:                pool.New().WithContext(ctx),
		questions:         make(map[int64]map[int64]chan telego.Message),
		callbackQuestions: make(map[int64]map[int64]chan telego.CallbackQuery),
	}
}

// Middleware
// This function handle users answers on questions
// If message == nil or message.From == nil middleware will call next function and don't call question callback
// If question for user created and message != nil and message.From != nil manager will call question callback
func (q *QuestionManager) Middleware(bot *telego.Bot, update telego.Update, next telegohandler.Handler) {
	if update.Message == nil {
		next(bot, update)

		return
	}

	if update.Message.From == nil {
		next(bot, update)

		return
	}

	chatID, userID := update.Message.Chat.ID, update.Message.From.ID

	chatQuestions, ok := q.questions[chatID]
	if !ok {
		next(bot, update)

		return
	}

	questionMessages, ok := chatQuestions[userID]
	if !ok {
		next(bot, update)

		return
	}

	message := *update.Message

	questionMessages <- message

	return
}

// NewQuestion
// This function create a new question from telego.Message
// If message.From == nil it will return ErrInvalidQuestionMessage
func (q *QuestionManager) NewQuestion(bot *telego.Bot, message telego.Message, callback callback) error {
	if message.From == nil {
		return ErrInvalidQuestionMessage
	}

	messages := make(chan telego.Message)

	chatID, userID := message.Chat.ID, message.From.ID

	q.mu.Lock()
	defer q.mu.Unlock()

	q.addQuestion(chatID, userID, messages)

	q.wg.Go(func(ctx context.Context) error {
		answer := func() (telego.Message, bool) {
			message, isOpen := <-messages

			return message, isOpen
		}

		callback(ctx, bot, answer)

		q.mu.Lock()
		defer q.mu.Unlock()

		q.deleteQuestion(chatID, userID)

		return nil
	})

	return nil
}

// NewCallbackQuestion
// This function create a new question from telego.CallbackQuery
// If message.Message == nil it will return ErrInvalidQuestionMessage
func (q *QuestionManager) NewCallbackQuestion(bot *telego.Bot, message telego.CallbackQuery, callback callback) error {
	if message.Message == nil {
		return ErrInvalidQuestionMessage
	}

	messages := make(chan telego.Message)

	chatID, userID := message.Message.Chat.ID, message.From.ID

	q.mu.Lock()
	defer q.mu.Unlock()

	q.addQuestion(chatID, userID, messages)

	q.wg.Go(func(ctx context.Context) error {
		answer := func() (telego.Message, bool) {
			message, isClosed := <-messages

			return message, isClosed
		}

		callback(ctx, bot, answer)

		q.mu.Lock()
		defer q.mu.Unlock()

		q.deleteQuestion(chatID, userID)

		return nil
	})

	return nil
}

// SetMaxGoroutines
// Set a max value for goroutines sync pool.
func (q *QuestionManager) SetMaxGoroutines(n int) {
	q.wg.WithMaxGoroutines(n)
}

func (q *QuestionManager) addQuestion(chatID, userID int64, messages chan telego.Message) {
	if _, ok := q.questions[chatID]; !ok {
		q.questions[chatID] = make(map[int64]chan telego.Message)
	}

	q.questions[chatID][userID] = messages
}

func (q *QuestionManager) deleteQuestion(chatID, userID int64) {
	defer func() {
		q.questions = copiedMap(q.questions)
	}()

	delete(q.questions[chatID], userID)

	if len(q.questions[chatID]) != 0 {
		return
	}

	delete(q.questions, chatID)
}
