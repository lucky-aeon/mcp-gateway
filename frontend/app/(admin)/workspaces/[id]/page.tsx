import { WorkspaceDetailPage } from '@/components/pages/workspace-detail-page'

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return <WorkspaceDetailPage workspaceId={id} />
}
