package coa

import "core:fmt"
import "core:strconv"
import "core:strings"
import slice "core:slice"

ValueKind :: enum {
    String,
    Number,
    List,
    Object,
    Typed,
}

ValueData :: union {
    string,
    f64,
    []Value,
    []Pair,
    TypedValue,
}

TypedValue :: struct {
    tag: string,
    value: ^Value,
}

Pair :: struct {
    key:   string,
    value: ^Value,
}

Value :: struct {
    kind: ValueKind,
    data: ValueData
}

Token_Kind :: enum {
    Identifier,
    String,
    Number,
    LBrace,
    RBrace,
    Equals,
    EOF,
    AtExpr,
}

Token :: struct {
    kind: Token_Kind,
    text: string,
    line: int,
    col:  int,
}

Parse_Error :: struct {
    message: string,
    line:    int,
    col:     int,
}

is_space :: proc(c: byte) -> bool {
    return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

is_ident_start :: proc(c: byte) -> bool {
    return (c >= 'a' && c <= 'z') ||
    (c >= 'A' && c <= 'Z') ||
    c == '_'
}

is_ident_continue :: proc(c: byte) -> bool {
    return (c >= 'a' && c <= 'z') ||
    (c >= 'A' && c <= 'Z') ||
    (c >= '0' && c <= '9') ||
    c == '_' || c == '/' || c == '.'
}

is_number_start :: proc(c: byte) -> bool {
    return (c >= '0' && c <= '9') || c == '-' || c == '+'
}

skip_to_eol :: proc(input: string, i: ^int, line: ^int, col: ^int) {
    for i^ < len(input) && input[i^] != '\n' {
        i^ += 1
        col^ += 1
    }
    if i^ < len(input) && input[i^] == '\n' {
        i^ += 1
        line^ += 1
        col^ = 1
    }
}

lex_at_expr :: proc(input: string, i: ^int, line: ^int, col: ^int) -> (string, bool) {
    start := i^

    if input[i^] != '@' {
        return "", false
    }

    i^ += 1
    col^ += 1

    if i^ < len(input) && input[i^] == '[' {
        i^ += 1
        col^ += 1

        depth := 1
        for i^ < len(input) && depth > 0 {
            ch := input[i^]

            if ch == '\n' {
                i^ += 1
                line^ += 1
                col^ = 1
                continue
            }

            if ch == '[' {
                depth += 1
            } else if ch == ']' {
                depth -= 1
            }

            i^ += 1
            col^ += 1
        }

        return input[start:i^], depth == 0
    }

    for i^ < len(input) && is_ident_continue(input[i^]) {
        i^ += 1
        col^ += 1
    }

    return input[start:i^], true
}

lex :: proc(input: string, allocator := context.allocator) -> ([]Token, bool) {
    tokens := make([dynamic]Token, 0, 0, allocator)
    i := 0
    line := 1
    col := 1

    advance :: proc(c: byte, i: ^int, line: ^int, col: ^int) {
        if c == '\n' {
            line^ += 1
            col^ = 1
        } else {
            col^ += 1
        }
        i^ += 1
    }
    for i < len(input) {
        c := input[i]

        if is_space(c) {
            advance(c, &i, &line, &col)
            continue
        }

        // Ignore directive/constant lines for now.
        if c == '#' {
            skip_to_eol(input, &i, &line, &col)
            continue
        }

        start_line := line
        start_col := col

        // Example: lines starting with '@'
        if c == '@' {
            text, ok := lex_at_expr(input, &i, &line, &col)
            if !ok {
                return nil, false
            }
            append(&tokens, Token{kind = .AtExpr, text = text, line = start_line, col = start_col})
            continue
        }

        switch c {
        case '{':
            append(&tokens, Token{.LBrace, "{", start_line, start_col})
            advance(c, &i, &line, &col)
        case '}':
            append(&tokens, Token{.RBrace, "}", start_line, start_col})
            advance(c, &i, &line, &col)
        case '=':
            append(&tokens, Token{.Equals, "=", start_line, start_col})
            advance(c, &i, &line, &col)
        case '"':
            start := i + 1
            advance(c, &i, &line, &col)

            builder := strings.builder_make(allocator)
            for i < len(input) {
                ch := input[i]
                if ch == '"' {
                    advance(ch, &i, &line, &col)
                    break
                }
                if ch == '\\' {
                    advance(ch, &i, &line, &col)
                    if i >= len(input) {
                        return nil, false
                    }
                    esc := input[i]
                    switch esc {
                    case 'n': strings.write_byte(&builder, '\n')
                    case 'r': strings.write_byte(&builder, '\r')
                    case 't': strings.write_byte(&builder, '\t')
                    case '\\': strings.write_byte(&builder, '\\')
                    case '"': strings.write_byte(&builder, '"')
                    case: strings.write_byte(&builder, esc)
                    }
                    advance(esc, &i, &line, &col)
                    continue
                }
                strings.write_byte(&builder, ch)
                advance(ch, &i, &line, &col)
            }
            append(&tokens, Token{.String, strings.to_string(builder), start_line, start_col})
            _ = start
        case:
            if is_ident_start(c) {
                start := i
                for i < len(input) && is_ident_continue(input[i]) {
                    advance(input[i], &i, &line, &col)
                }
                append(&tokens, Token{.Identifier, input[start:i], start_line, start_col})
            } else if is_number_start(c) {
                start := i
                dot_count := 0
                if input[i] == '+' || input[i] == '-' {
                    advance(input[i], &i, &line, &col)
                }
                for i < len(input) {
                    ch := input[i]
                    if ch >= '0' && ch <= '9' {
                        advance(ch, &i, &line, &col)
                    } else if ch == '.' {
                        dot_count += 1
                        if dot_count > 1 {
                            break
                        }
                        advance(ch, &i, &line, &col)
                    } else {
                        break
                    }
                }
                append(&tokens, Token{.Number, input[start:i], start_line, start_col})
            } else {
                return nil, false
            }
        }
    }

    append(&tokens, Token{.EOF, "", line, col})
    return tokens[:], true
}

Parser :: struct {
    tokens: []Token,
    pos:    int,
    err:    Parse_Error,
    has_err: bool,
}

current :: proc(p: ^Parser) -> Token {
    return p.tokens[p.pos]
}

peek :: proc(p: ^Parser, offset: int) -> Token {
    idx := p.pos + offset
    if idx >= len(p.tokens) {
        return p.tokens[len(p.tokens)-1]
    }
    return p.tokens[idx]
}

advance_tok :: proc(p: ^Parser) -> Token {
    t := p.tokens[p.pos]
    if p.pos < len(p.tokens)-1 {
        p.pos += 1
    }
    return t
}

fail :: proc(p: ^Parser, msg: string) {
    t := current(p)
    p.err = Parse_Error{msg, t.line, t.col}
    p.has_err = true
}

expect_one_of :: proc(p: ^Parser, kinds: []Token_Kind) -> (Token, bool) {
    t := current(p)
    if !slice.contains(kinds, t.kind) {
        fail(p, fmt.tprintf("expected %v, got %v", kinds, t.kind))
        return Token{}, false
    }
    advance_tok(p)
    return t, true
}

expect :: proc(p: ^Parser, kind: Token_Kind) -> (Token, bool) {
    t := current(p)
    if t.kind != kind {
        fail(p, fmt.tprintf("expected %v, got %v", kind, t.kind))
        return Token{}, false
    }
    advance_tok(p)
    return t, true
}

parse_value :: proc(p: ^Parser, allocator := context.allocator) -> (Value, bool) {
    t := current(p)

    switch t.kind {
    case .String:
        advance_tok(p)
        return Value{kind = .String, data = t.text}, true
    case .Identifier:
        if peek(p, 1).kind == .LBrace {
            tag := t.text
            advance_tok(p)

            inner, ok := parse_braced(p, allocator)
            if !ok {
                return Value{}, false
            }

            return Value{
                kind = .Typed,
                data = TypedValue{
                    tag = tag,
                    value = new_clone(inner, allocator),
                },
            }, true
        }
        advance_tok(p)
        return Value{kind = .String, data = t.text}, true
    case .Number:
        advance_tok(p)
        n, ok := strconv.parse_f64(t.text)
        if !ok {
            fail(p, "invalid number")
            return Value{}, false
        }
        return Value{kind = .Number, data = n}, true
    case .LBrace:
        return parse_braced(p, allocator)
    case .AtExpr:
        advance_tok(p)
        return Value{kind = .Number, data = f64(0)}, true

    //TODO
    case .RBrace:
    case .Equals:
        return Value{}, false
    case .EOF:

    }
    fail(p, "unexpected token while parsing value")
    return Value{}, false
}


parse_braced :: proc(p: ^Parser, allocator := context.allocator) -> (Value, bool) {
    _, ok := expect(p, .LBrace)
    if !ok {
        return Value{}, false
    }

    if current(p).kind == .RBrace {
        advance_tok(p)
        empty := make([]Pair, 0, allocator)
        return Value{kind = .Object, data = empty}, true
    }

    if current(p).kind == .Identifier && peek(p, 1).kind == .Equals {
        items := make([dynamic]Pair, 0, 0, allocator)
        for current(p).kind != .RBrace && current(p).kind != .EOF {
            key_tok, ok := expect_one_of(p, {.Identifier, .AtExpr})
            if !ok {
                return Value{}, false
            }
            _, ok = expect(p, .Equals)
            if !ok {
                return Value{}, false
            }
            val, valueOk := parse_value(p, allocator)
            if !valueOk {
                return Value{}, false
            }
            append(&items, Pair{key_tok.text, new_clone(val, allocator)})
        }
        _, ok = expect(p, .RBrace)
        if !ok {
            return Value{}, false
        }
        return Value{kind = .Object, data = items[:]}, true
    }

    items := make([dynamic]Value, 0, 0, allocator)
    for current(p).kind != .RBrace && current(p).kind != .EOF {
        val, ok := parse_value(p, allocator)
        if !ok {
            return Value{}, false
        }
        append(&items, val)
    }
    _, ok = expect(p, .RBrace)
    if !ok {
        return Value{}, false
    }
    return Value{kind = .List, data = items[:]}, true
}

Parse_Result :: struct {
    root: Value,
    err:  Parse_Error,
    ok:   bool,
}

parse_document :: proc(input: string, allocator := context.allocator) -> Parse_Result {
    tokens, ok := lex(input, allocator)
    if !ok {
        return Parse_Result{
            ok = false,
            err = Parse_Error{"lexing failed", 0, 0},
        }
    }

    p := Parser{tokens = tokens}
    items := make([dynamic]Pair, 0, 0, allocator)

    for current(&p).kind != .EOF && !p.has_err {
        key_tok, ok := expect_one_of(&p, {.Identifier, .AtExpr})
        if !ok {
            break
        }
        _, ok = expect(&p, .Equals)
        if !ok {
            break
        }
        val, valueOk := parse_value(&p, allocator)
        if !valueOk {
            break
        }
        append(&items, Pair{key_tok.text, new_clone(val, allocator)})
    }

    if p.has_err {
        return Parse_Result{ok = false, err = p.err}
    }

    root := Value{kind = .Object, data = items[:]}
    return Parse_Result{root = root, ok = true}
}

main :: proc() {
    input := `@semy = 0.25
@third = @[1/3]
@fifth = @[1/5]
@sixth = @[1/6]

GBR2_communist_canton = {
    pattern = "pattern_solid.tga"
    color1 = rgb { 5 12 5 }
}`

    result := parse_document(input)

    if !result.ok {
        fmt.eprintln("Parse error:", result.err.message, "at", result.err.line, ":", result.err.col)
        return
    }

    fmt.printfln("%v", result)
}