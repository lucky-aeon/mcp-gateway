import { toast } from '@/hooks/use-toast'
import { GatewayApiError } from '@/lib/gateway-api'

type ActionFeedbackOptions = {
  successTitle?: string
  successDescription?: string
  errorTitle?: string
  errorDescription?: string
}

export async function runAction(
  action: () => Promise<unknown>,
  options: ActionFeedbackOptions = {}
): Promise<boolean> {
  try {
    await action()
    if (options.successTitle) {
      toast({
        title: options.successTitle,
        description: options.successDescription,
      })
    }
    return true
  } catch (error) {
    toast({
      variant: 'destructive',
      title: options.errorTitle || '操作失败',
      description: actionErrorMessage(error, options.errorDescription),
    })
    return false
  }
}

export function actionErrorMessage(error: unknown, fallback = '请求失败，请稍后重试') {
  if (error instanceof GatewayApiError) {
    return error.message
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}
