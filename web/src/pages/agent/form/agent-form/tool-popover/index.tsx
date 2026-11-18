import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AccessFeature } from '@/constants/access-profile';
import { useIsAccessFeatureEnabled } from '@/hooks/use-access-profile-request';
import { Operator } from '@/pages/agent/constant';
import { AgentFormContext, AgentInstanceContext } from '@/pages/agent/context';
import useGraphStore from '@/pages/agent/store';
import { Position } from '@xyflow/react';
import { t } from 'i18next';
import { useContext, useEffect } from 'react';
import { useGetAgentMCPIds, useGetAgentToolNames } from '../use-get-tools';
import { MCPCommand, ToolCommand } from './tool-command';
import { useUpdateAgentNodeMCP } from './use-update-mcp';

enum ToolType {
  Common = 'common',
  MCP = 'mcp',
}

export function ToolPopover({ children }: React.PropsWithChildren) {
  const { addCanvasNode } = useContext(AgentInstanceContext);
  const node = useContext(AgentFormContext);
  const { toolNames } = useGetAgentToolNames();
  const deleteAgentToolNodeById = useGraphStore(
    (state) => state.deleteAgentToolNodeById,
  );
  const { mcpIds } = useGetAgentMCPIds();
  const { updateNodeMCP } = useUpdateAgentNodeMCP();
  const mcpEnabled = useIsAccessFeatureEnabled(AccessFeature.Mcp);

  useEffect(() => {
    const total = toolNames.length + mcpIds.length;
    if (node?.id) {
      if (total > 0) {
        addCanvasNode(Operator.Tool, {
          position: Position.Bottom,
          nodeId: node?.id,
        })();
      } else {
        deleteAgentToolNodeById(node.id);
      }
    }
  }, [
    addCanvasNode,
    deleteAgentToolNodeById,
    mcpIds.length,
    node?.id,
    toolNames.length,
  ]);

  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className="w-80 p-4">
        <Tabs defaultValue={ToolType.Common}>
          <TabsList>
            <TabsTrigger value={ToolType.Common}>
              {t('flow.builtIn')}
            </TabsTrigger>
            {mcpEnabled && <TabsTrigger value={ToolType.MCP}>MCP</TabsTrigger>}
          </TabsList>
          <TabsContent value={ToolType.Common}>
            <ToolCommand />
          </TabsContent>
          {mcpEnabled && (
            <TabsContent value={ToolType.MCP}>
              <MCPCommand value={mcpIds} onChange={updateNodeMCP}></MCPCommand>
            </TabsContent>
          )}
        </Tabs>
      </PopoverContent>
    </Popover>
  );
}
