package printer

import (
	"go/ast"
	"go/token"
	"strconv"
)

// Map html-style to actual js event names
var eventMap = map[string]string{
	"onAbort":          "abort",
	"onCancel":         "cancel",
	"onCanPlay":        "canplay",
	"onCanPlaythrough": "canplaythrough",
	"onChange":         "change",
	"onClick":          "click",
	"onCueChange":      "cuechange",
	"onDblClick":       "dblclick",
	"onDurationChange": "durationchange",
	"onEmptied":        "emptied",
	"onEnded":          "ended",
	"onInput":          "input",
	"onInvalid":        "invalid",
	"onKeyDown":        "keydown",
	"onKeyPress":       "keypress",
	"onKeyUp":          "keyup",
	"onLoadedData":     "loadeddata",
	"onLoadedMetadata": "loadedmetadata",
	"onLoadStart":      "loadstart",
	"onMouseDown":      "mousedown",
	"onMouseEnter":     "mouseenter",
	"onMouseleave":     "mouseleave",
	"onMouseMove":      "mousemove",
	"onMouseOut":       "mouseout",
	"onMouseOver":      "mouseover",
	"onMouseUp":        "mouseup",
	"onMouseWheel":     "mousewheel",
	"onPause":          "pause",
	"onPlay":           "play",
	"onPlaying":        "playing",
	"onProgress":       "progress",
	"onRateChange":     "ratechange",
	"onReset":          "reset",
	"onSeeked":         "seeked",
	"onSeeking":        "seeking",
	"onSelect":         "select",
	"onShow":           "show",
	"onStalled":        "stalled",
	"onSubmit":         "submit",
	"onSuspend":        "suspend",
	"onTimeUpdate":     "timeupdate",
	"onToggle":         "toggle",
	"onVolumeChange":   "volumechange",
	"onWaiting":        "waiting",
}

var attrMap = map[string]string{
	"autofocus":   "autofocus",
	"checked":     "checked",
	"class":       "class",
	"for":         "htmlFor",
	"href":        "href",
	"id":          "id",
	"placeholder": "placeholder",
	"src":         "src",
	"type":        "type",
	"value":       "value",
}

func goxToVecty(genname string, gox *ast.GoxExpr) ast.Expr {
	isComponent := ast.IsGoxComponent(gox.TagName)

	if isComponent {
		return newComponent(genname, gox)
	}

	tagName := gox.TagName.(*ast.Ident).Name
	tagFn := "Tag"

	var args []ast.Expr
	if tagName == "text" {
		tagFn = "PlainText"
	} else {
		args = []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(tagName),
			}}
	}

	if len(gox.Attrs) > 0 {
		// Create markup expr and add attributes
		markup := newCallExpr(
			newSelectorExpr(genname, "Markup"),
			mapProps(genname, gox.Attrs),
		)

		// Add the markup
		args = append(args, markup)
	}

	args = append(args, xToArgs(genname, gox.X)...)

	return newCallExpr(
		newSelectorExpr(genname, tagFn),
		args,
	)
}

func xToArgs(genname string, x []ast.Expr) []ast.Expr {
	args := make([]ast.Expr, len(x))
	// Add the contents
	for i, expr := range x {
		switch expr := expr.(type) {
		// TODO figure out what's a better thing to do here
		// do we want to error on compile or figure out what to do based on context?
		// (I think the latter)
		// Fallback to regular behavior, don't wrap this yet
		//case *ast.GoExpr:
		//	e := newCallExpr(
		//		newSelectorExpr(genname, "Text"),
		//		[]ast.Expr{expr},
		//	)
		//	args = append(args, e)

		case *ast.BareWordsExpr:
			// if len(strings.TrimSpace(expr.Value)) == 0 {
			// 	continue
			// }
			e := newCallExpr(
				newSelectorExpr(genname, "Text"),
				[]ast.Expr{expr},
			)
			args[i] = e
		case *ast.GoExpr:
			e := newCallExpr(
				newSelectorExpr(genname, "Value"),
				[]ast.Expr{expr},
			)
			args[i] = e
		default:
			args[i] = expr
		}
	}

	return args
}

func newSelectorExpr(x, sel string) *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X:   ast.NewIdent(x),
		Sel: ast.NewIdent(sel)}
}

func newCallExpr(fun ast.Expr, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:      fun,
		Args:     args,
		Ellipsis: token.NoPos, Lparen: token.NoPos, Rparen: token.NoPos}
}

func newComponent(genname string, gox *ast.GoxExpr) ast.Expr {
	if _, ok := gox.TagName.(*ast.CallExpr); ok {
		return newComponentCall(genname, gox)
	}

	return newComponentStruct(genname, gox)
}

func newComponentStruct(genname string, gox *ast.GoxExpr) ast.Expr {
	args := make([]ast.Expr, len(gox.Attrs))
	for i, attr := range gox.Attrs {
		if attr.Rhs == nil { // default to true like JSX
			attr.Rhs = ast.NewIdent("true")
		}
		expr := &ast.KeyValueExpr{
			Key:   ast.NewIdent(attr.Lhs.Name),
			Colon: token.NoPos,
			Value: attr.Rhs,
		}

		args[i] = expr
	}

	if len(gox.X) != 0 {
		expr := &ast.KeyValueExpr{
			Key:   ast.NewIdent("Body"),
			Colon: token.NoPos,
			Value: newCallExpr(
				newSelectorExpr(genname, "Writers"),
				xToArgs(genname, gox.X),
			),
		}

		args = append(args, expr)
	}

	return newCallExpr(
		newSelectorExpr(genname, "NewComponent"),
		[]ast.Expr{
			&ast.UnaryExpr{
				OpPos: token.NoPos,
				Op:    token.AND,
				X: &ast.CompositeLit{
					Type:   gox.TagName,
					Lbrace: token.NoPos,
					Elts:   args,
					Rbrace: token.NoPos,
				},
			},
		},
	)
}

func newComponentCall(genname string, gox *ast.GoxExpr) ast.Expr {
	attrs := make([]ast.Expr, len(gox.Attrs))
	for i, attr := range gox.Attrs {
		if attr.Rhs == nil { // default to true like JSX
			attr.Rhs = ast.NewIdent("true")
		}

		expr := newCallExpr(
			newSelectorExpr(genname, "Property"),
			[]ast.Expr{
				&ast.BasicLit{
					ValuePos: attr.Lhs.Pos(),
					Kind:     token.STRING,
					Value:    strconv.Quote(attr.Lhs.Name),
				},
				attr.Rhs,
			},
		)

		attrs[i] = expr
	}

	args := []ast.Expr{}
	if len(attrs) > 0 {
		args = append(args, newCallExpr(
			newSelectorExpr(genname, "Markup"),
			attrs,
		))
	}

	if len(gox.X) != 0 {
		expr := newCallExpr(
			newSelectorExpr(genname, "Writers"),
			xToArgs(genname, gox.X),
		)

		args = append(args, expr)
	}

	t := gox.TagName.(*ast.CallExpr)

	if len(gox.X) != 0 || len(attrs) != 0 {
		t.Args = append(t.Args, args...)
		// t.Args = append(t.Args, newCallExpr(
		// 	newSelectorExpr(genname, "Text"),
		// 	append([]ast.Expr{
		// 		&ast.BasicLit{
		// 			ValuePos: token.NoPos,
		// 			Value:    `""`,
		// 			Kind:     token.STRING,
		// 		},
		// 	}, args...),
		// ))
	}
	return t
}

func mapProps(genname string, goxAttrs []*ast.GoxAttrStmt) []ast.Expr {
	var mapped = []ast.Expr{}
	for _, attr := range goxAttrs {
		// set default of Rhs to true if none provided
		if attr.Rhs == nil { // default to true like JSX
			attr.Rhs = ast.NewIdent("true")
		}

		var expr ast.Expr

		// if prop is an event listener (e.g. "onClick")
		if _, ok := eventMap[attr.Lhs.Name]; ok {
			expr = newEventListener(genname, attr)
		} else if mappedName, ok := attrMap[attr.Lhs.Name]; ok {
			// if it's a vecty controlled prop
			expr = newCallExpr(
				newSelectorExpr(genname, "Property"),
				[]ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(mappedName)},
					attr.Rhs,
				},
			)
		} else if attr.Lhs.Name == "attrs" {
			expr = attr.Rhs
		} else {
			// if prop is a normal attribute
			expr = newCallExpr(
				newSelectorExpr(genname, "Attribute"),
				[]ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(attr.Lhs.Name)},
					attr.Rhs,
				},
			)
		}

		mapped = append(mapped, expr)
	}

	return mapped
}

func newEventListener(genname string, goxAttr *ast.GoxAttrStmt) ast.Expr {
	return &ast.UnaryExpr{
		OpPos: token.NoPos,
		Op:    token.AND,
		X: &ast.CompositeLit{
			Type:   newSelectorExpr(genname, "EventListener"),
			Lbrace: token.NoPos,
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key: ast.NewIdent("Name"),
					Value: &ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(eventMap[goxAttr.Lhs.Name]),
					},
				},
				&ast.KeyValueExpr{
					Key:   ast.NewIdent("Listener"),
					Value: goxAttr.Rhs,
				},
			},
			Rbrace: token.NoPos,
		},
	}
}
