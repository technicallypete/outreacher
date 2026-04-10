"use client";
/// <reference types="@assistant-ui/core/store" />

import React, { type FC, type ReactNode, useMemo, useState, useCallback, useContext, createContext, useRef } from "react";
import {
  AssistantRuntimeProvider,
  ThreadPrimitive,
  ComposerPrimitive,
  MessagePrimitive,
  AttachmentPrimitive,
  SimpleTextAttachmentAdapter,
  SimpleImageAttachmentAdapter,
  CompositeAttachmentAdapter,
  ThreadListPrimitive,
  ThreadListItemPrimitive,
  useThread,
  useMessage,
  useComposer,
} from "@assistant-ui/react";
import {
  useRemoteThreadListRuntime,
  RuntimeAdapterProvider,
  useComposerAddAttachment,
} from "@assistant-ui/core/react";
import type { ThreadHistoryAdapter, RemoteThreadListAdapter } from "@assistant-ui/core";
import { useAISDKRuntime, AssistantChatTransport } from "@assistant-ui/react-ai-sdk";
import { useAui } from "@assistant-ui/store";
import { useChat } from "@ai-sdk/react";
import { createAssistantStream } from "assistant-stream";
import { MarkdownTextPrimitive } from "@assistant-ui/react-markdown";
import remarkGfm from "remark-gfm";
import type {
  TextMessagePartComponent,
  ToolCallMessagePartComponent,
} from "@assistant-ui/core/react";
import { ChevronLeft, ChevronRight, Paperclip, SendHorizonal, StopCircle, X } from "lucide-react";

// ── Attachment adapter ────────────────────────────────────────────────────────

const textAdapter = new SimpleTextAttachmentAdapter();
// Windows assigns application/vnd.ms-excel to .csv files
textAdapter.accept += ",application/vnd.ms-excel,.csv";

const attachmentAdapter = new CompositeAttachmentAdapter([
  textAdapter,
  new SimpleImageAttachmentAdapter(),
]);

// ── Per-thread history provider ───────────────────────────────────────────────

// Injected as unstable_Provider on the thread list adapter.
// Runs inside ThreadListItemRuntimeProvider so useAui() has thread context.
const ThreadHistoryProvider: FC<{ children: ReactNode }> = ({ children }) => {
  const aui = useAui();

  const historyAdapter = useMemo<ThreadHistoryAdapter>(
    () => ({
      // These are unused — useAISDKRuntime only calls withFormat()
      load: async () => ({ messages: [] }),
      append: async () => {},

      withFormat(formatAdapter) {
        return {
          async load() {
            const remoteId = aui.threadListItem().getState().remoteId;
            if (!remoteId) return { headId: null, messages: [] };
            const res = await fetch(`/api/threads/${remoteId}/messages`);
            if (!res.ok) return { headId: null, messages: [] };
            const data = await res.json() as {
              headId: string | null;
              messages: Array<{ id: string; parent_id: string | null; format: string; content: Record<string, unknown> }>;
            };
            return {
              headId: data.headId,
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              messages: data.messages.map((row) => formatAdapter.decode(row as any)),
            };
          },
          async append({ message, parentId }) {
            const { remoteId } = await aui.threadListItem().initialize();
            const encoded = formatAdapter.encode({ message, parentId });
            const id = formatAdapter.getId(message);
            await fetch(`/api/threads/${remoteId}/messages`, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ id, parentId, content: encoded }),
            });
          },
        };
      },
    }),
    [aui]
  );

  return (
    <RuntimeAdapterProvider adapters={{ history: historyAdapter }}>
      {children}
    </RuntimeAdapterProvider>
  );
};

// ── Remote thread list adapter ────────────────────────────────────────────────

const threadListAdapter: RemoteThreadListAdapter = {
  async list() {
    const res = await fetch("/api/threads");
    if (!res.ok) return { threads: [] };
    const { threads } = await res.json() as {
      threads: Array<{ id: string; title: string | null; status: string }>;
    };
    return {
      threads: threads.map((t) => ({
        remoteId: t.id,
        externalId: undefined,
        title: t.title ?? undefined,
        status: t.status as "regular" | "archived",
      })),
    };
  },

  async initialize(_threadId) {
    const res = await fetch("/api/threads", { method: "POST" });
    const { id } = await res.json() as { id: string };
    return { remoteId: id, externalId: undefined };
  },

  async fetch(threadId) {
    const res = await fetch(`/api/threads/${threadId}`);
    if (!res.ok) throw new Error("Thread not found");
    const t = await res.json() as { id: string; title: string | null; status: string };
    return {
      remoteId: t.id,
      externalId: undefined,
      title: t.title ?? undefined,
      status: t.status as "regular" | "archived",
    };
  },

  async rename(remoteId, newTitle) {
    await fetch(`/api/threads/${remoteId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: newTitle }),
    });
  },

  async archive(remoteId) {
    await fetch(`/api/threads/${remoteId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status: "archived" }),
    });
  },

  async unarchive(remoteId) {
    await fetch(`/api/threads/${remoteId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status: "regular" }),
    });
  },

  async delete(remoteId) {
    await fetch(`/api/threads/${remoteId}`, { method: "DELETE" });
  },

  async generateTitle(remoteId, messages) {
    // Derive title from first user message, persist it, stream it back
    const firstUser = messages.find((m) => m.role === "user");
    const title =
      firstUser?.content
        .filter((p): p is { type: "text"; text: string } => p.type === "text")
        .map((p) => p.text)
        .join(" ")
        .slice(0, 60) ?? "New Chat";

    // Persist asynchronously (no need to await)
    fetch(`/api/threads/${remoteId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title }),
    }).catch(() => {});

    return createAssistantStream((ctrl) => {
      ctrl.appendText(title);
      ctrl.close();
    });
  },

  unstable_Provider: ThreadHistoryProvider,
};

// ── Chat error context ────────────────────────────────────────────────────────

const ChatErrorContext = createContext<{
  error: string | null;
  setError: (e: string | null) => void;
}>({ error: null, setError: () => {} });

// ── Runtime hook (one instance per active thread) ─────────────────────────────

function useThreadRuntime() {
  const { setError } = useContext(ChatErrorContext);
  const [transport] = useState(
    () => new AssistantChatTransport({ api: "/api/chat" })
  );
  const chat = useChat({
    transport,
    onError: (err) => {
      const msg = (err as { message?: string }).message ?? String(err);
      const isOverLimit = msg.includes("413") || msg.includes("too large") || msg.includes("exceeded");
      setError(isOverLimit
        ? "File too large to send. Try importing fewer rows at a time."
        : "Something went wrong. Please try again.");
    },
  });
  return useAISDKRuntime(chat, { adapters: { attachments: attachmentAdapter } });
}

// ── Root component ────────────────────────────────────────────────────────────

export default function ChatPanel() {
  const [error, setError] = useState<string | null>(null);

  const runtime = useRemoteThreadListRuntime({
    adapter: threadListAdapter,
    runtimeHook: useThreadRuntime,
  });

  const [sidebarOpen, setSidebarOpen] = useState(true);
  const toggleSidebar = useCallback(() => setSidebarOpen((v) => !v), []);

  return (
    <ChatErrorContext.Provider value={{ error, setError }}>
      <AssistantRuntimeProvider runtime={runtime}>
        <div className="flex h-full overflow-hidden w-full max-w-5xl border-l border-gray-200 dark:border-gray-800">
          <div className="flex-1 flex flex-col min-w-0">
            <Thread />
          </div>
          <Sidebar open={sidebarOpen} onToggle={toggleSidebar} />
        </div>
      </AssistantRuntimeProvider>
    </ChatErrorContext.Provider>
  );
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

function Sidebar({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  return (
    <div className={`${open ? "w-60" : "w-10"} shrink-0 border-l border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-900 flex flex-col h-full overflow-hidden transition-[width] duration-200`}>
      <div className="p-2 border-b border-gray-200 dark:border-gray-800 shrink-0 flex items-center gap-2">
        <button
          onClick={onToggle}
          className="shrink-0 flex items-center justify-center w-6 h-6 rounded-md text-gray-500 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
          aria-label={open ? "Collapse sidebar" : "Expand sidebar"}
        >
          {open ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
        </button>
        {open && (
          <ThreadListPrimitive.New asChild>
            <button className="flex-1 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-1.5 text-sm text-left text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
              + New Chat
            </button>
          </ThreadListPrimitive.New>
        )}
      </div>
      {open && (
        <div className="flex-1 overflow-y-auto py-1">
          <ThreadListPrimitive.Items components={{ ThreadListItem }} />
        </div>
      )}
    </div>
  );
}

function ThreadListItem() {
  return (
    <ThreadListItemPrimitive.Root className="group flex items-center gap-1 px-2 py-1.5 rounded-lg mx-1 my-0.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 data-[active]:bg-blue-50 dark:data-[active]:bg-blue-950 data-[active]:text-blue-900 dark:data-[active]:text-blue-300 transition-colors">
      <ThreadListItemPrimitive.Trigger className="flex-1 text-left truncate min-w-0 cursor-pointer">
        <ThreadListItemPrimitive.Title fallback="New Chat" />
      </ThreadListItemPrimitive.Trigger>
      <ThreadListItemPrimitive.Delete asChild>
        <button className="hidden group-hover:flex shrink-0 items-center p-0.5 text-gray-400 hover:text-red-500 transition-colors">
          <X className="h-3.5 w-3.5" />
        </button>
      </ThreadListItemPrimitive.Delete>
    </ThreadListItemPrimitive.Root>
  );
}

// ── Thread UI ─────────────────────────────────────────────────────────────────

function Thread() {
  const { error, setError } = useContext(ChatErrorContext);
  return (
    <ThreadPrimitive.Root className="flex flex-col h-full bg-slate-50 dark:bg-gray-600">
      <ThreadPrimitive.Viewport className="flex-1 overflow-y-auto px-4 py-6 space-y-6">
        <ThreadPrimitive.Empty>
          <div className="flex flex-col items-center justify-center h-full text-center text-gray-400 dark:text-gray-500 pt-20">
            <p className="text-lg font-medium text-gray-600 dark:text-gray-300 mb-1">
              Outreacher Assistant
            </p>
            <p className="text-sm max-w-xs dark:text-gray-400">
              Search leads, import contacts, update statuses, or add follow-up
              notes.
            </p>
          </div>
        </ThreadPrimitive.Empty>

        <ThreadPrimitive.Messages
          components={{ UserMessage, AssistantMessage }}
        />
      </ThreadPrimitive.Viewport>

      {error && (
        <div className="mx-4 mb-2 flex items-start gap-2 rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950 px-3 py-2 text-sm text-red-700 dark:text-red-300">
          <span className="flex-1">{error}</span>
          <button
            onClick={() => setError(null)}
            className="shrink-0 text-red-400 hover:text-red-600 dark:hover:text-red-200"
            aria-label="Dismiss"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      <div className="border-t border-gray-200 dark:border-gray-800 bg-slate-50 dark:bg-gray-700 px-4 py-3">
        <Composer />
      </div>
    </ThreadPrimitive.Root>
  );
}

function UserMessage() {
  return (
    <MessagePrimitive.Root className="flex justify-end mb-4">
      <div className="max-w-[75%] bg-blue-600 text-white rounded-2xl rounded-br-sm px-4 py-2 text-sm shadow-md">
        <MessagePrimitive.Parts components={{ Text: UserText }} />
      </div>
    </MessagePrimitive.Root>
  );
}

// Matches <attachment name="foo.csv">...content...</attachment>
const ATTACHMENT_RE = /<attachment name="?([^">]+)"?>([\s\S]*?)<\/attachment>/g;

const UserText: TextMessagePartComponent = ({ text }) => {
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});

  const parts: React.ReactNode[] = [];
  let last = 0;
  let idx = 0;
  let match: RegExpExecArray | null;
  ATTACHMENT_RE.lastIndex = 0;

  while ((match = ATTACHMENT_RE.exec(text)) !== null) {
    const [full, name, content] = match;
    if (match.index > last) {
      parts.push(<span key={`t${idx}`} className="whitespace-pre-wrap">{text.slice(last, match.index)}</span>);
    }
    const i = idx;
    const rows = content.trim().split("\n").filter(Boolean);
    const dataRows = Math.max(rows.length - 1, 0);
    parts.push(
      <span key={`a${i}`} className="block mt-1">
        <span className="inline-flex items-center gap-1 bg-blue-700 rounded px-2 py-0.5 text-xs font-medium">
          📎 {name}
        </span>
        {" "}
        <button
          onClick={() => setExpanded((e) => ({ ...e, [i]: !e[i] }))}
          className="text-xs text-blue-200 underline underline-offset-2 hover:text-white"
        >
          {expanded[i] ? "hide" : `${dataRows} rows`}
        </button>
        {expanded[i] && (
          <span className="block mt-1 whitespace-pre-wrap text-xs opacity-80">{content.trim()}</span>
        )}
      </span>
    );
    last = match.index + full.length;
    idx++;
  }

  if (last < text.length) {
    parts.push(<span key={`t${idx}`} className="whitespace-pre-wrap">{text.slice(last)}</span>);
  }

  if (parts.length === 0) return <span className="whitespace-pre-wrap">{text}</span>;
  return <>{parts}</>;
};

function AssistantMessage() {
  return (
    <MessagePrimitive.Root className="flex justify-start mb-4">
      <div className="max-w-[85%] bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-2xl rounded-bl-sm px-4 py-3 text-sm text-gray-800 dark:text-gray-100 shadow-md">
        <AssistantMessageContent />
      </div>
    </MessagePrimitive.Root>
  );
}

function AssistantMessageContent() {
  const hasText = useMessage(
    (m) => m.content.some((p) => p.type === "text" && (p as { text: string }).text.length > 0)
  );
  return (
    <>
      {!hasText && <span className="text-gray-400 dark:text-gray-500">…</span>}
      <MessagePrimitive.Parts
        components={{ Text: AssistantText, tools: { Fallback: ToolCallDisplay } }}
      />
    </>
  );
}

const AssistantText: TextMessagePartComponent = () => (
  <MarkdownTextPrimitive
    remarkPlugins={[remarkGfm]}
    className="prose prose-sm dark:prose-invert max-w-none prose-table:border-collapse prose-td:border prose-td:border-gray-200 dark:prose-td:border-gray-600 prose-td:px-3 prose-td:py-1.5 prose-th:border prose-th:border-gray-200 dark:prose-th:border-gray-600 prose-th:px-3 prose-th:py-1.5 prose-th:bg-gray-50 dark:prose-th:bg-gray-700 prose-th:font-medium"
  />
);

const ToolCallDisplay: ToolCallMessagePartComponent = ({ toolName, status }) => {
  const label = toolName.replace(/_/g, " ");
  const running =
    "type" in status &&
    (status.type === "running" || status.type === "requires-action");

  return (
    <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 py-1 my-1">
      {running ? (
        <span className="inline-block w-3 h-3 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
      ) : (
        <span className="text-green-500">✓</span>
      )}
      <span>{running ? `Running ${label}…` : `Used ${label}`}</span>
    </div>
  );
};

// ── Composer ──────────────────────────────────────────────────────────────────

const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024; // 10 MB

function AttachButton() {
  const { addAttachment } = useComposerAddAttachment();
  const { setError } = useContext(ChatErrorContext);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    if (file.size > MAX_ATTACHMENT_BYTES) {
      setError(`"${file.name}" is too large (${(file.size / 1024 / 1024).toFixed(1)} MB). Maximum is 10 MB.`);
      return;
    }
    await addAttachment(file);
  };

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        accept={attachmentAdapter.accept}
        className="hidden"
        onChange={handleChange}
      />
      <button
        type="button"
        onClick={() => inputRef.current?.click()}
        className="shrink-0 rounded-lg p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
      >
        <Paperclip className="h-4 w-4" />
      </button>
    </>
  );
}

function SendButton() {
  const isRunning = useThread((t) => t.isRunning);
  const oversizedFile = useComposer((c) => {
    for (const a of c.attachments) {
      const f = (a as { file?: File }).file;
      if (f && f.size > MAX_ATTACHMENT_BYTES) return f.name;
    }
    return null;
  });

  if (isRunning) {
    return (
      <ComposerPrimitive.Cancel asChild>
        <button className="shrink-0 rounded-lg bg-blue-600 p-1.5 text-white hover:bg-blue-700 transition-colors">
          <span className="block w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
        </button>
      </ComposerPrimitive.Cancel>
    );
  }

  if (oversizedFile) {
    return (
      <button
        disabled
        title={`"${oversizedFile}" exceeds the 10 MB limit — remove it to send`}
        className="shrink-0 rounded-lg bg-blue-600 p-1.5 text-white opacity-40 cursor-not-allowed transition-colors"
      >
        <SendHorizonal className="h-4 w-4" />
      </button>
    );
  }

  return (
    <ComposerPrimitive.Send asChild>
      <button className="shrink-0 rounded-lg bg-blue-600 p-1.5 text-white hover:bg-blue-700 disabled:opacity-40 transition-colors">
        <SendHorizonal className="h-4 w-4" />
      </button>
    </ComposerPrimitive.Send>
  );
}

function Composer() {
  return (
    <ComposerPrimitive.Root className="rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 focus-within:border-blue-500 dark:focus-within:border-blue-400 focus-within:ring-1 focus-within:ring-blue-500 dark:focus-within:ring-blue-400 transition-colors">
      <ComposerPrimitive.Attachments>
        {({ attachment }) => (
          <AttachmentPrimitive.Root className="flex items-center gap-1.5 m-2 mb-0 px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded-lg text-xs text-gray-700 dark:text-gray-300 max-w-[200px]">
            <span className="truncate flex-1">
              <AttachmentPrimitive.Name />
            </span>
            <AttachmentPrimitive.Remove asChild>
              <button className="shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200">
                <X className="h-3 w-3" />
              </button>
            </AttachmentPrimitive.Remove>
          </AttachmentPrimitive.Root>
        )}
      </ComposerPrimitive.Attachments>

      <div className="flex items-end gap-2 px-3 py-2">
        <AttachButton />

        <ComposerPrimitive.Input
          className="flex-1 resize-none bg-transparent text-sm text-gray-900 dark:text-gray-100 outline-none placeholder:text-gray-400 dark:placeholder:text-gray-500 max-h-40 min-h-[24px]"
          placeholder="Ask anything about your leads…"
          autoFocus
        />
        <SendButton />
        <ComposerPrimitive.Cancel asChild>
          <button className="shrink-0 rounded-lg bg-gray-100 dark:bg-gray-700 p-1.5 text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors">
            <StopCircle className="h-4 w-4" />
          </button>
        </ComposerPrimitive.Cancel>
      </div>
    </ComposerPrimitive.Root>
  );
}
