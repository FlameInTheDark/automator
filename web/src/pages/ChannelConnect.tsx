import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { MessageSquare, CheckCircle2 } from 'lucide-react'

import { api } from '../api/client'
import { Card, CardContent } from '../components/ui/Card'
import Button from '../components/ui/Button'
import Input from '../components/ui/Input'

export default function ChannelConnect() {
  const [searchParams] = useSearchParams()
  const [code, setCode] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const channelId = searchParams.get('channelId') || undefined

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code.trim()) return

    setIsSubmitting(true)
    setError(null)

    try {
      await api.channels.connect({
        channel_id: channelId,
        code: code.trim(),
      })
      setSuccess(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect channel')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-bg flex items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardContent className="p-8">
          <div className="text-center mb-6">
            <div className="w-14 h-14 rounded-2xl bg-blue-500/10 flex items-center justify-center mx-auto mb-4">
              {success ? (
                <CheckCircle2 className="w-7 h-7 text-green-400" />
              ) : (
                <MessageSquare className="w-7 h-7 text-blue-400" />
              )}
            </div>
            <h1 className="text-xl font-semibold text-text">
              {success ? 'Channel Connected' : 'Connect Channel'}
            </h1>
            <p className="text-sm text-text-muted mt-2">
              {success
                ? 'This chat is now linked. You can go back to your messenger and continue.'
                : 'Enter the one-time code from Telegram or Discord to connect this chat.'}
            </p>
          </div>

          {!success && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <Input
                  value={code}
                  onChange={(e) => setCode(e.target.value.toUpperCase())}
                  placeholder="Enter code"
                  className="text-center font-mono tracking-[0.2em]"
                  maxLength={6}
                />
              </div>
              {error && (
                <p className="text-sm text-red-400 text-center">{error}</p>
              )}
              <Button type="submit" className="w-full" loading={isSubmitting} disabled={!code.trim()}>
                Connect
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
