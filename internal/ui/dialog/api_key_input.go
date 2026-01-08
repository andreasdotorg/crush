package dialog

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

type APIKeyInputState int

const (
	APIKeyInputStateInitial APIKeyInputState = iota
	APIKeyInputStateVerifying
	APIKeyInputStateVerified
	APIKeyInputStateError
)

type APIKeyStateChangeMsg struct {
	State APIKeyInputState
}

// APIKeyInputID is the identifier for the model selection dialog.
const APIKeyInputID = "api_key_input"

// APIKeyInput represents a model selection dialog.
type APIKeyInput struct {
	com *common.Common

	provider catwalk.Provider
	modelID  string

	width int
	state APIKeyInputState

	keyMap struct {
		Submit key.Binding
		Close  key.Binding
	}
	input   textinput.Model
	spinner spinner.Model
	help    help.Model
}

var _ Dialog = (*APIKeyInput)(nil)

// NewAPIKeyInput creates a new Models dialog.
func NewAPIKeyInput(com *common.Common, provider catwalk.Provider, modelID string) (*APIKeyInput, error) {
	t := com.Styles

	m := APIKeyInput{}
	m.com = com
	m.provider = provider
	m.modelID = modelID
	m.width = 60

	m.input = textinput.New()
	m.input.SetVirtualCursor(false)
	m.input.Placeholder = "Enter you API key..."
	m.input.SetStyles(com.Styles.TextInput)
	m.input.Focus()
	m.input.SetWidth(m.width - t.Dialog.InputPrompt.GetHorizontalFrameSize() - 1) // (1) cursor padding

	m.spinner = spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(t.Base.Foreground(t.Green)),
	)

	m.help = help.New()
	m.help.Styles = t.DialogHelpStyles()

	m.keyMap.Submit = key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "submit"),
	)
	m.keyMap.Close = CloseKey

	return &m, nil
}

// ID implements Dialog.
func (m *APIKeyInput) ID() string {
	return APIKeyInputID
}

// Update implements tea.Model.
func (m *APIKeyInput) Update(msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case APIKeyStateChangeMsg:
		m.state = msg.State
		switch m.state {
		case APIKeyInputStateVerifying:
			return tea.Batch(
				m.spinner.Tick,
				m.verifyAPIKey,
			)()
		}
	case spinner.TickMsg:
		switch m.state {
		case APIKeyInputStateVerifying:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			if cmd != nil {
				return cmd()
			}
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keyMap.Close):
			return CloseMsg{}
		case key.Matches(msg, m.keyMap.Submit):
			switch m.state {
			case APIKeyInputStateInitial, APIKeyInputStateError:
				return APIKeyStateChangeMsg{State: APIKeyInputStateVerifying}
			case APIKeyInputStateVerified:
				return CloseMsg{}
			case APIKeyInputStateVerifying:
				// do nothing
			}
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			if cmd != nil {
				return cmd()
			}
		}
	case tea.PasteMsg:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if cmd != nil {
			return cmd()
		}
	}
	return nil
}

// View implements tea.Model.
func (m *APIKeyInput) View() string {
	t := m.com.Styles

	textStyle := t.Dialog.SecondaryText
	helpStyle := t.Dialog.HelpView
	dialogStyle := t.Dialog.View.Width(m.width)
	inputStyle := t.Dialog.InputPrompt
	helpStyle = helpStyle.Width(m.width - dialogStyle.GetHorizontalFrameSize())

	m.input.Prompt = m.spinner.View()

	content := strings.Join([]string{
		m.headerView(),
		inputStyle.Render(m.inputView()),
		textStyle.Render("This will be written in your global configuration:"),
		textStyle.Render(config.GlobalConfigData()),
		"",
		helpStyle.Render(m.help.View(m)),
	}, "\n")

	return dialogStyle.Render(content)
}

func (m *APIKeyInput) headerView() string {
	t := m.com.Styles
	titleStyle := t.Dialog.Title
	dialogStyle := t.Dialog.View.Width(m.width)

	headerOffset := titleStyle.GetHorizontalFrameSize() + dialogStyle.GetHorizontalFrameSize()
	return common.DialogTitle(t, titleStyle.Render(m.dialogTitle()), m.width-headerOffset)
}

func (m *APIKeyInput) dialogTitle() string {
	t := m.com.Styles
	textStyle := t.Dialog.TitleText
	errorStyle := t.Dialog.TitleError
	accentStyle := t.Dialog.TitleAccent

	switch m.state {
	case APIKeyInputStateInitial:
		return textStyle.Render("Enter your ") + accentStyle.Render(fmt.Sprintf("%s Key", m.provider.Name)) + textStyle.Render(".")
	case APIKeyInputStateVerifying:
		return textStyle.Render("Verifying your ") + accentStyle.Render(fmt.Sprintf("%s Key", m.provider.Name)) + textStyle.Render("...")
	case APIKeyInputStateVerified:
		return accentStyle.Render(fmt.Sprintf("%s Key", m.provider.Name)) + textStyle.Render(" validated.")
	case APIKeyInputStateError:
		return errorStyle.Render("Invalid ") + accentStyle.Render(fmt.Sprintf("%s Key", m.provider.Name)) + errorStyle.Render(". Try again?")
	}
	return ""
}

func (m *APIKeyInput) inputView() string {
	switch m.state {
	case APIKeyInputStateInitial:
		m.input.Prompt = "> "
	case APIKeyInputStateVerifying:
		m.input.Prompt = m.spinner.View()
	case APIKeyInputStateVerified:
		m.input.Prompt = styles.CheckIcon + " "
	case APIKeyInputStateError:
		m.input.Prompt = styles.ErrorIcon + " "
	}
	return m.input.View()
}

// FullHelp returns the full help view.
func (m *APIKeyInput) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			m.keyMap.Submit,
			m.keyMap.Close,
		},
	}
}

// ShortHelp returns the full help view.
func (m *APIKeyInput) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keyMap.Submit,
		m.keyMap.Close,
	}
}

func (m *APIKeyInput) verifyAPIKey() tea.Msg {
	start := time.Now()

	providerConfig := config.ProviderConfig{
		ID:      string(m.provider.ID),
		Name:    m.provider.Name,
		APIKey:  m.input.Value(),
		Type:    m.provider.Type,
		BaseURL: m.provider.APIEndpoint,
	}
	err := providerConfig.TestConnection(config.Get().Resolver())

	// intentionally wait for at least 750ms to make sure the user sees the spinner
	elapsed := time.Since(start)
	minimum := 750 * time.Millisecond
	if elapsed < minimum {
		time.Sleep(minimum - elapsed)
	}

	if err == nil {
		return APIKeyStateChangeMsg{APIKeyInputStateVerified}
	}
	return APIKeyStateChangeMsg{APIKeyInputStateError}
}
