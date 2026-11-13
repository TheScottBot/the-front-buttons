package editor

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
)

// An app screen is a page of one application's keys, opened from an app key
// with App screen ticked and left from its top left tile, the application's
// icon. The editor lists app screens under the regular pages, numbers only the
// regular pages and pages through them alone, as the deck does. On a screen of
// an application with bricks of its own, such as GoLand, those bricks and
// their groups are offered beside every other brick, and nowhere else.

// appSetOf is the bricks of a page's own application, if it is an app screen
// for one that has them.
func (pages *pageHandlers) appSetOf(page config.Page) (brick.AppSet, bool) {
	if !page.IsAppScreen() || pages.options.Registry == nil {
		return brick.AppSet{ID: "", Name: "", Identifiers: nil, Groups: nil}, false
	}

	return pages.options.Registry.AppSetFor(page.Application)
}

// offeredOn reports whether a type choice may be used on a page: any preset
// or general brick, and an application's own bricks on its screen only.
func (pages *pageHandlers) offeredOn(resolved keyType, page config.Page) bool {
	if resolved.presetName != "" {
		return true
	}

	kind, found := pages.options.Registry.Kind(resolved.kindName)
	if !found || kind.Scope() == "" {
		return found
	}

	set, hasSet := pages.appSetOf(page)

	return hasSet && set.ID == kind.Scope()
}

// pageAt is the page at an index, or an empty regular page for one that does
// not exist, which offers the general bricks.
func pageAt(configuration config.Configuration, pageIndex int) config.Page {
	if pageIndex < 0 || pageIndex >= len(configuration.Pages) {
		return config.Page{Name: "", Application: "", Keys: nil}
	}

	return configuration.Pages[pageIndex]
}

// regularPages lists the regular pages by their index in the file, in order.
func regularPages(allPages []config.Page) []int {
	regular := make([]int, 0, len(allPages))

	for pageIndex, page := range allPages {
		if !page.IsAppScreen() {
			regular = append(regular, pageIndex)
		}
	}

	return regular
}

// screenTitle names an app screen by its application's bricks, as GoLand, or
// otherwise by the page's own name.
func (pages *pageHandlers) screenTitle(page config.Page) string {
	if set, hasSet := pages.appSetOf(page); hasSet {
		return set.Name
	}

	return page.Name
}

// appScreenIndex finds the screen of an application.
func appScreenIndex(allPages []config.Page, application string) (int, bool) {
	for pageIndex, page := range allPages {
		if page.IsAppScreen() && page.Application == application {
			return pageIndex, true
		}
	}

	return 0, false
}

// wantedScreen is the application whose screen a key asks for, if it asks.
func wantedScreen(parameters any) (string, bool) {
	source, offers := parameters.(brick.ApplicationScreenSource)
	if !offers {
		return "", false
	}

	return source.ApplicationScreen()
}

// wantedScreenPage is the screen a key asks for, as a new page named after
// the application, if it asks for one.
func (pages *pageHandlers) wantedScreenPage(requestContext context.Context, parameters any) (config.Page, bool) {
	application, wanted := wantedScreen(parameters)
	if !wanted {
		return config.Page{Name: "", Application: "", Keys: nil}, false
	}

	return config.Page{
		Name: pages.applicationName(requestContext, application), Application: application, Keys: nil,
	}, true
}

// addScreen adds an app screen unless its application has one already.
func addScreen(configuration *config.Configuration, screen config.Page) {
	if _, exists := appScreenIndex(configuration.Pages, screen.Application); !exists {
		configuration.Pages = append(configuration.Pages, screen)
	}
}

// resolveOn resolves a type choice that may be used on a page.
func (pages *pageHandlers) resolveOn(choice string, page config.Page) (keyType, error) {
	resolved, typeError := pages.resolveType(choice)
	if typeError == nil && !pages.offeredOn(resolved, page) {
		typeError = fmt.Errorf("%w: %q is not offered on this page", errUnknownType, choice)
	}

	return resolved, typeError
}

// applicationName is what a new app screen is called: the name of the
// application's own bricks, as GoLand; or its name in the app picker; or the
// last part of its identifier. It is cut to a page name's length.
func (pages *pageHandlers) applicationName(requestContext context.Context, application string) string {
	name := ""

	if set, hasSet := pages.options.Registry.AppSetFor(application); hasSet {
		name = set.Name
	}

	for _, choice := range pages.apps.current(requestContext) {
		if name == "" && choice.Value == application {
			name = choice.Label
		}
	}

	if name == "" {
		name = identifierName(application)
	}

	runes := []rune(name)
	if len(runes) > config.MaximumPageNameCharacters {
		runes = runes[:config.MaximumPageNameCharacters]
	}

	return string(runes)
}

// identifierName is the readable end of an application's identifier: Safari
// from com.apple.Safari, goland64 from a Windows path to goland64.exe.
func identifierName(application string) string {
	name := path.Base(strings.ReplaceAll(application, `\`, "/"))
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".exe"), ".app")

	if dot := strings.LastIndex(name, "."); dot >= 0 && dot < len(name)-1 {
		name = name[dot+1:]
	}

	return name
}

// kindTitle is what the editor calls an application's own brick: the label
// the key shows until given its own, or else the kind's name.
func kindTitle(kind brick.Kind) string {
	constructed, constructError := kind.Construct(kind.Schema().NewParameters(), brick.Ports{
		URLOpener: nil, ApplicationLauncher: nil, PipelineProvider: nil, EditorAddress: nil, KeyPresser: nil,
		MediaPlayer: nil, WindowCycler: nil, ShortcutPresser: nil,
	})
	if labelled, isLabelled := constructed.(brick.DefaultLabelled); isLabelled && constructError == nil {
		return labelled.DefaultLabel()
	}

	return kind.Name()
}

// appTypeOptions are a page's own application's bricks, for the key form and
// the drawer, named by what each does.
func (pages *pageHandlers) appTypeOptions(page config.Page, selected string) []typeOption {
	set, hasSet := pages.appSetOf(page)
	if !hasSet {
		return nil
	}

	kindNames := pages.options.Registry.KindNamesIn(set.ID)
	options := make([]typeOption, 0, len(kindNames))

	for _, kindName := range kindNames {
		kind, _ := pages.options.Registry.Kind(kindName)
		value := kindChoicePrefix + kindName
		options = append(options, typeOption{
			Value: value, Label: set.Name + ": " + kindTitle(kind), Selected: value == selected,
		})
	}

	return options
}

// appGroupView is one of an application's groups in the drawer.
type appGroupView struct {
	Name   string
	Label  string
	Width  int
	Height int
}

// appGroupName is what the drawer and the group form call an application's
// group: the set's name and the group's, joined by a dot.
func appGroupName(set brick.AppSet, group brick.AppGroup) string {
	return set.ID + "." + group.Name
}

// appGroupViews are a page's own application's groups, for the drawer.
func (pages *pageHandlers) appGroupViews(page config.Page) []appGroupView {
	set, hasSet := pages.appSetOf(page)
	if !hasSet {
		return nil
	}

	views := make([]appGroupView, 0, len(set.Groups))

	for _, group := range set.Groups {
		views = append(views, appGroupView{
			Name:  appGroupName(set, group),
			Label: fmt.Sprintf("%dx%d %s", group.Width, group.Height, group.Label),
			Width: group.Width, Height: group.Height,
		})
	}

	return views
}

// appGroups adds every application's groups to the drawer's own, by name,
// with a part for each of their bricks.
func (pages *pageHandlers) appGroups(sizes map[string]groupSize, parts map[string]groupPart) {
	if pages.options.Registry == nil {
		return
	}

	for _, set := range pages.options.Registry.AppSets() {
		for _, group := range set.Groups {
			sizes[appGroupName(set, group)] = groupSize{width: group.Width, height: group.Height, parts: group.Kinds}

			for _, kindName := range group.Kinds {
				kind, _ := pages.options.Registry.Kind(kindName)
				parts[kindName] = groupPart{
					name: kindName, choice: kindChoicePrefix + kindName, title: kindTitle(kind), label: "", icon: "",
				}
			}
		}
	}
}
