<?php

declare(strict_types=1);

namespace App;

enum Keyword: string
{
    case ABSTRACT = 'abstract';
    case AND = 'and';
    case ARRAY = 'array';
    case AS = 'as';
    case BREAK = 'break';
    case CALLABLE = 'callable';
    case CASE = 'case';
    case CATCH = 'catch';
    case CLONE = 'clone';
    case CONST = 'const';
    case CONTINUE = 'continue';
    case DECLARE = 'declare';
    case DEFAULT = 'default';
    case DIE = 'die';
    case DO = 'do';
    case ECHO = 'echo';
    case ELSE = 'else';
    case ELSEIF = 'elseif';
    case EMPTY = 'empty';
    case ENDDECLARE = 'enddeclare';
    case ENDFOR = 'endfor';
    case ENDFOREACH = 'endforeach';
    case ENDIF = 'endif';
    case ENDSWITCH = 'endswitch';
    case ENDWHILE = 'endwhile';
    case ENUM = 'enum';
    case EVAL = 'eval';
    case EXIT = 'exit';
    case EXTENDS = 'extends';
    case FINAL = 'final';
    case FINALLY = 'finally';
    case FN = 'fn';
    case FOR = 'for';
    case FOREACH = 'foreach';
    case FUNCTION = 'function';
    case GLOBAL = 'global';
    case GOTO = 'goto';
    case IF = 'if';
    case IMPLEMENTS = 'implements';
    case INCLUDE = 'include';
    case INCLUDE_ONCE = 'include_once';
    case INSTANCEOF = 'instanceof';
    case INSTEADOF = 'insteadof';
    case INTERFACE = 'interface';
    case ISSET = 'isset';
    case LIST = 'list';
    case MATCH = 'match';
    case NAMESPACE = 'namespace';
    case NEW = 'new';
    case OR = 'or';
    case PRINT = 'print';
    case PRIVATE = 'private';
    case PROTECTED = 'protected';
    case PUBLIC = 'public';
    case READONLY = 'readonly';
    case REQUIRE = 'require';
    case REQUIRE_ONCE = 'require_once';
    case RETURN = 'return';
    case STATIC = 'static';
    case SWITCH = 'switch';
    case THROW = 'throw';
    case TRAIT = 'trait';
    case TRY = 'try';
    case UNSET = 'unset';
    case USE = 'use';
    case VAR = 'var';
    case WHILE = 'while';
    case XOR = 'xor';
    case YIELD = 'yield';
    case __CLASS__ = '__class__';
    case __DIR__ = '__dir__';
    case __FILE__ = '__file__';
    case __FUNCTION__ = '__function__';
    case __LINE__ = '__line__';
    case __METHOD__ = '__method__';
    case __NAMESPACE__ = '__namespace__';
    case __PROPERTY__ = '__property__';
    case __TRAIT__ = '__trait__';

    public function isLoop(): bool
    {
        return match ($this) {
            self::DO, self::FOR, self::FOREACH, self::WHILE => true,
            default => false,
        };
    }
}

enum Pure
{
    case Abstract;
    case And;
    case Array;
    case As;
    case Break;
    case Callable;
    case Case;
    case Catch;
    case Clone;
    case Const;
    case Continue;
    case Declare;
    case Default;
    case Die;
    case Do;
    case Echo;
    case Else;
    case Elseif;
    case Empty;
    case Enddeclare;
    case Endfor;
    case Endforeach;
    case Endif;
    case Endswitch;
    case Endwhile;
    case Enum;
    case Eval;
    case Exit;
    case Extends;
    case Final;
    case Finally;
    case Fn;
    case For;
    case Foreach;
    case Function;
    case Global;
    case Goto;
    case If;
    case Implements;
    case Include;
    case Include_once;
    case Instanceof;
    case Insteadof;
    case Interface;
    case Isset;
    case List;
    case Match;
    case Namespace;
    case New;
    case Or;
    case Print;
    case Private;
    case Protected;
    case Public;
    case Readonly;
    case Require;
    case Require_once;
    case Return;
    case Static;
    case Switch;
    case Throw;
    case Trait;
    case Try;
    case Unset;
    case Use;
    case Var;
    case While;
    case Xor;
    case Yield;
    case __class__;
    case __dir__;
    case __file__;
    case __function__;
    case __line__;
    case __method__;
    case __namespace__;
    case __property__;
    case __trait__;
}

interface Typed
{
    public const string ABSTRACT = 'abstract';

    public const string AND = 'and';

    public const string ARRAY = 'array';

    public const string AS = 'as';

    public const string BREAK = 'break';

    public const string CALLABLE = 'callable';

    public const string CASE = 'case';

    public const string CATCH = 'catch';

    public const string CLONE = 'clone';

    public const string CONST = 'const';

    public const string CONTINUE = 'continue';

    public const string DECLARE = 'declare';

    public const string DEFAULT = 'default';

    public const string DIE = 'die';

    public const string DO = 'do';

    public const string ECHO = 'echo';

    public const string ELSE = 'else';

    public const string ELSEIF = 'elseif';

    public const string EMPTY = 'empty';

    public const string ENDDECLARE = 'enddeclare';

    public const string ENDFOR = 'endfor';

    public const string ENDFOREACH = 'endforeach';

    public const string ENDIF = 'endif';

    public const string ENDSWITCH = 'endswitch';

    public const string ENDWHILE = 'endwhile';

    public const string ENUM = 'enum';

    public const string EVAL = 'eval';

    public const string EXIT = 'exit';

    public const string EXTENDS = 'extends';

    public const string FINAL = 'final';

    public const string FINALLY = 'finally';

    public const string FN = 'fn';

    public const string FOR = 'for';

    public const string FOREACH = 'foreach';

    public const string FUNCTION = 'function';

    public const string GLOBAL = 'global';

    public const string GOTO = 'goto';

    public const string IF = 'if';

    public const string IMPLEMENTS = 'implements';

    public const string INCLUDE = 'include';

    public const string INCLUDE_ONCE = 'include_once';

    public const string INSTANCEOF = 'instanceof';

    public const string INSTEADOF = 'insteadof';

    public const string INTERFACE = 'interface';

    public const string ISSET = 'isset';

    public const string LIST = 'list';

    public const string MATCH = 'match';

    public const string NAMESPACE = 'namespace';

    public const string NEW = 'new';

    public const string OR = 'or';

    public const string PRINT = 'print';

    public const string PRIVATE = 'private';

    public const string PROTECTED = 'protected';

    public const string PUBLIC = 'public';

    public const string READONLY = 'readonly';

    public const string REQUIRE = 'require';

    public const string REQUIRE_ONCE = 'require_once';

    public const string RETURN = 'return';

    public const string STATIC = 'static';

    public const string SWITCH = 'switch';

    public const string THROW = 'throw';

    public const string TRAIT = 'trait';

    public const string TRY = 'try';

    public const string UNSET = 'unset';

    public const string USE = 'use';

    public const string VAR = 'var';

    public const string WHILE = 'while';

    public const string XOR = 'xor';

    public const string YIELD = 'yield';

    public const string __CLASS__ = '__class__';

    public const string __DIR__ = '__dir__';

    public const string __FILE__ = '__file__';

    public const string __FUNCTION__ = '__function__';

    public const string __LINE__ = '__line__';

    public const string __METHOD__ = '__method__';

    public const string __NAMESPACE__ = '__namespace__';

    public const string __PROPERTY__ = '__property__';

    public const string __TRAIT__ = '__trait__';
}

final class Usage
{
    public const ABSTRACT = 'abstract';

    public const AND = 'and';

    public const ARRAY = 'array';

    public const AS = 'as';

    public const BREAK = 'break';

    public const CALLABLE = 'callable';

    public const CASE = 'case';

    public const CATCH = 'catch';

    public const CLONE = 'clone';

    public const CONST = 'const';

    public const CONTINUE = 'continue';

    public const DECLARE = 'declare';

    public const DEFAULT = 'default';

    public const DIE = 'die';

    public const DO = 'do';

    public const ECHO = 'echo';

    public const ELSE = 'else';

    public const ELSEIF = 'elseif';

    public const EMPTY = 'empty';

    public const ENDDECLARE = 'enddeclare';

    public const ENDFOR = 'endfor';

    public const ENDFOREACH = 'endforeach';

    public const ENDIF = 'endif';

    public const ENDSWITCH = 'endswitch';

    public const ENDWHILE = 'endwhile';

    public const ENUM = 'enum';

    public const EVAL = 'eval';

    public const EXIT = 'exit';

    public const EXTENDS = 'extends';

    public const FINAL = 'final';

    public const FINALLY = 'finally';

    public const FN = 'fn';

    public const FOR = 'for';

    public const FOREACH = 'foreach';

    public const FUNCTION = 'function';

    public const GLOBAL = 'global';

    public const GOTO = 'goto';

    public const IF = 'if';

    public const IMPLEMENTS = 'implements';

    public const INCLUDE = 'include';

    public const INCLUDE_ONCE = 'include_once';

    public const INSTANCEOF = 'instanceof';

    public const INSTEADOF = 'insteadof';

    public const INTERFACE = 'interface';

    public const ISSET = 'isset';

    public const LIST = 'list';

    public const MATCH = 'match';

    public const NAMESPACE = 'namespace';

    public const NEW = 'new';

    public const OR = 'or';

    public const PRINT = 'print';

    public const PRIVATE = 'private';

    public const PROTECTED = 'protected';

    public const PUBLIC = 'public';

    public const READONLY = 'readonly';

    public const REQUIRE = 'require';

    public const REQUIRE_ONCE = 'require_once';

    public const RETURN = 'return';

    public const STATIC = 'static';

    public const SWITCH = 'switch';

    public const THROW = 'throw';

    public const TRAIT = 'trait';

    public const TRY = 'try';

    public const UNSET = 'unset';

    public const USE = 'use';

    public const VAR = 'var';

    public const WHILE = 'while';

    public const XOR = 'xor';

    public const YIELD = 'yield';

    public const __CLASS__ = '__class__';

    public const __DIR__ = '__dir__';

    public const __FILE__ = '__file__';

    public const __FUNCTION__ = '__function__';

    public const __LINE__ = '__line__';

    public const __METHOD__ = '__method__';

    public const __NAMESPACE__ = '__namespace__';

    public const __PROPERTY__ = '__property__';

    public const __TRAIT__ = '__trait__';

    public function run(): string
    {
        return Keyword::DO->value . self::IF . Pure::For->name . Typed::NEW;
    }
}
