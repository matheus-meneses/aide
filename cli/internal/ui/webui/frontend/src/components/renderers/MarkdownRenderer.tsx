import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { handleExternalClick } from "@/lib/openExternal";

interface Props {
  content: string;
  onSuggestionClick?: (text: string) => void;
}

function sanitizeHref(href?: string): string | undefined {
  if (!href) return undefined;
  try {
    const url = new URL(href, window.location.origin);
    if (url.protocol === "http:" || url.protocol === "https:" || url.protocol === "mailto:") {
      return href;
    }
  } catch {
    // invalid URL
  }
  return undefined;
}

function extractSuggestions(content: string): { body: string; suggestions: string[] } {
  const lines = content.trim().split("\n");
  const suggestions: string[] = [];
  while (lines.length > 0) {
    const last = lines[lines.length - 1].trim();
    if (/^(Would you|Do you|Should I|Can I|Want me)/.test(last)) {
      suggestions.unshift(lines.pop() ?? "");
    } else break;
  }
  return { body: lines.join("\n"), suggestions };
}

export function MarkdownRenderer({ content, onSuggestionClick }: Props) {
  const { body, suggestions } = extractSuggestions(content);

  const components: Components = {
    a: ({ href, children }) => {
      const safeHref = sanitizeHref(href);
      if (!safeHref) {
        return <span className="underline text-muted-foreground">{children}</span>;
      }
      return (
        <a
          href={safeHref}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => handleExternalClick(e, safeHref)}
          className="underline text-info"
        >
          {children}
        </a>
      );
    },
    table: ({ children }) => (
      <div className="my-2 overflow-x-auto rounded border">
        <table className="w-full text-sm">{children}</table>
      </div>
    ),
    thead: ({ children }) => <thead className="bg-accent/50">{children}</thead>,
    th: ({ children }) => (
      <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">{children}</th>
    ),
    td: ({ children }) => <td className="px-3 py-2 border-t">{children}</td>,
    tr: ({ children }) => <tr className="even:bg-accent/20">{children}</tr>,
    hr: () => <hr className="my-4 border-border" />,
    code: ({ className, children }) => {
      const isBlock = className?.includes("language-");
      if (isBlock) {
        return (
          <code className={`${className} block bg-muted rounded p-3 text-xs overflow-x-auto`}>
            {children}
          </code>
        );
      }
      return <code className="px-1 py-0.5 bg-muted rounded text-xs">{children}</code>;
    },
    pre: ({ children }) => <pre className="my-2">{children}</pre>,
  };

  return (
    <div>
      <div className="prose prose-sm dark:prose-invert max-w-none [&_p]:my-1 [&_ul]:my-1 [&_li]:my-0 [&_ol]:my-1">
        <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
          {body}
        </ReactMarkdown>
      </div>
      {suggestions.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-3 pt-2 border-t border-border/50">
          {suggestions.map((s, i) => (
            <button
              key={i}
              onClick={() => onSuggestionClick?.(s)}
              className="px-2.5 py-1 text-xs rounded-full border bg-accent/30 hover:bg-accent transition-colors text-left"
            >
              {s}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
