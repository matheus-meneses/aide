import {type FormEvent, type KeyboardEvent, type RefObject} from 'react'
import {Send, Square} from 'lucide-react'
import {type SlashCommand} from '@/lib/commands'
import {CommandPalette} from './CommandPalette'

interface Props {
    input: string
    setInput: (value: string) => void
    isStreaming: boolean
    inputRef: RefObject<HTMLTextAreaElement>
    onSubmit: (e: FormEvent) => void
    onCancel: () => void
    onKeyDown: (e: KeyboardEvent) => void
    showCommands: boolean
    filteredCommands: SlashCommand[]
    selectedIdx: number
    onSelectCommand: (name: string) => void
}

export function ChatComposer({
    input,
    setInput,
    isStreaming,
    inputRef,
    onSubmit,
    onCancel,
    onKeyDown,
    showCommands,
    filteredCommands,
    selectedIdx,
    onSelectCommand,
}: Props) {
    return (
        <form onSubmit={onSubmit} className="border-t p-3 relative">
            {showCommands && (
                <CommandPalette
                    commands={filteredCommands}
                    selectedIdx={selectedIdx}
                    onSelect={onSelectCommand}
                />
            )}
            <div className="flex items-end gap-2 max-w-3xl mx-auto">
          <textarea
              ref={inputRef}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={onKeyDown}
              placeholder="Ask something or type / for commands..."
              rows={1}
              disabled={isStreaming}
              className="flex-1 resize-none rounded-lg border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 min-h-[38px] max-h-[120px] disabled:opacity-50 disabled:cursor-not-allowed"
              style={{height: 'auto', overflow: 'hidden'}}
              onInput={e => {
                  const t = e.currentTarget
                  t.style.height = 'auto'
                  t.style.height = Math.min(t.scrollHeight, 120) + 'px'
              }}
          />
                {isStreaming ? (
                    <button
                        type="button"
                        onClick={onCancel}
                        className="p-2 rounded-lg bg-red-500/10 text-red-500 hover:bg-red-500/20 transition-colors"
                        aria-label="Stop generating"
                    >
                        <Square className="w-4 h-4"/>
                    </button>
                ) : (
                    <button
                        type="submit"
                        disabled={!input.trim()}
                        className="p-2 rounded-lg bg-primary text-primary-foreground disabled:opacity-40 hover:opacity-90 transition-opacity"
                        aria-label="Send message"
                    >
                        <Send className="w-4 h-4"/>
                    </button>
                )}
            </div>
        </form>
    )
}
