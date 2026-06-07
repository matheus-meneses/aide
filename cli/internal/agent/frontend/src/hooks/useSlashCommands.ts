import {useEffect, useState} from 'react'
import {COMMANDS, execCommand, generateHelpText, parseCommand} from '@/lib/commands'
import {type Message} from '@/hooks/useChatStream'

interface Params {
    input: string
    injectMessage: (msg: Message) => void
    clearMessages: () => void
    markAtBottom: () => void
}

export function useSlashCommands({input, injectMessage, clearMessages, markAtBottom}: Params) {
    const [showCommands, setShowCommands] = useState(false)
    const [commandFilter, setCommandFilter] = useState('')
    const [selectedIdx, setSelectedIdx] = useState(0)

    const filteredCommands = COMMANDS.filter(c =>
        commandFilter === '' || c.name.startsWith(commandFilter)
    )

    useEffect(() => {
        if (input === '/' || (input.startsWith('/') && !input.includes(' '))) {
            setShowCommands(true)
            setCommandFilter(input.slice(1))
            setSelectedIdx(0)
        } else {
            setShowCommands(false)
        }
    }, [input])

    const handleSlashCommand = async (text: string) => {
        const {name, args} = parseCommand(text)
        const fullCommand = args ? `${name} ${args}` : name

        const userMsg: Message = {id: `cmd-${Date.now()}`, role: 'user', content: text, timestamp: Date.now()}
        injectMessage(userMsg)
        markAtBottom()

        if (name === 'clear') {
            clearMessages()
            return
        }

        if (name === 'help') {
            const helpMsg: Message = {
                id: `help-${Date.now()}`,
                role: 'assistant',
                content: generateHelpText(),
                timestamp: Date.now(),
                format: 'text'
            }
            injectMessage(helpMsg)
            return
        }

        try {
            const result = await execCommand(fullCommand)
            const responseMsg: Message = {
                id: `exec-${Date.now()}`,
                role: 'assistant',
                content: result.text || '',
                timestamp: Date.now(),
                format: result.type,
                data: result.data,
            }
            injectMessage(responseMsg)
        } catch (err: any) {
            injectMessage({
                id: `exec-err-${Date.now()}`,
                role: 'assistant',
                content: `Command failed: ${err?.message || 'network error'}`,
                timestamp: Date.now(),
                format: 'text',
                isError: true,
            })
        }
    }

    return {
        showCommands,
        setShowCommands,
        commandFilter,
        selectedIdx,
        setSelectedIdx,
        filteredCommands,
        handleSlashCommand,
    }
}
