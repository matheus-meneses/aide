import {type SlashCommand} from '@/lib/commands'

interface Props {
    commands: SlashCommand[]
    selectedIdx: number
    onSelect: (name: string) => void
}

export function CommandPalette({commands, selectedIdx, onSelect}: Props) {
    if (commands.length === 0) return null

    return (
        <div className="absolute bottom-full left-0 right-0 mx-3 mb-1">
            <div className="max-w-3xl mx-auto bg-card border rounded-lg shadow-lg overflow-hidden">
                {commands.map((cmd, i) => (
                    <button
                        key={cmd.name}
                        type="button"
                        onClick={() => onSelect(cmd.name)}
                        className={`w-full flex items-center gap-3 px-3 py-2 text-left text-sm transition-colors ${
                            i === selectedIdx ? 'bg-accent' : 'hover:bg-accent/50'
                        }`}
                    >
                        <span className="font-mono text-xs text-primary">/{cmd.name}</span>
                        <span className="text-muted-foreground text-xs">{cmd.description}</span>
                    </button>
                ))}
            </div>
        </div>
    )
}
