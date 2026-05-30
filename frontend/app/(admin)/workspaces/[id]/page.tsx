import { WorkspaceDetailPage } from '@/components/pages/workspace-detail-page'

export default function Page({ params }: { params: { id: string } }) {
  return <WorkspaceDetailPage workspaceId={params.id} />
}
