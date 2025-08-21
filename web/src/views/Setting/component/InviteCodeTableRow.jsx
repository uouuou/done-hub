import PropTypes from 'prop-types'
import { useEffect, useState } from 'react'
import { Checkbox, IconButton, TableCell, TableRow, Tooltip } from '@mui/material'
import { Icon } from '@iconify/react'
import Label from 'ui-component/Label'
import TableSwitch from 'ui-component/Switch'
import { copy, showError, showSuccess, timestamp2string } from 'utils/common'
import { API } from 'utils/api'
import { useTranslation } from 'react-i18next'

// 常量定义 - 与主组件保持一致
const STATUS = {
  ENABLED: 1,
  DISABLED: 2
}

export default function InviteCodeTableRow({ item, selected, onSelectRow, onRefresh, handleOpenModal }) {
  const { t } = useTranslation()
  const [statusSwitch, setStatusSwitch] = useState(item.status)
  const [statusLoading, setStatusLoading] = useState(false)

  // 当item.status变化时，同步更新本地状态
  useEffect(() => {
    setStatusSwitch(item.status)
  }, [item.status])

  const handleStatus = async() => {
    if (statusLoading) return

    setStatusLoading(true)
    try {
      const newStatus = statusSwitch === STATUS.ENABLED
        ? STATUS.DISABLED
        : STATUS.ENABLED

      const res = await API.put(`/api/invite-code/${item.id}`, {
        name: item.name,
        max_uses: item.max_uses,
        status: newStatus,
        starts_at: item.starts_at,
        expires_at: item.expires_at
      })

      const { success, message } = res.data
      if (success) {
        setStatusSwitch(newStatus)
        showSuccess(newStatus === STATUS.ENABLED ? '邀请码已启用' : '邀请码已禁用')
        onRefresh()
      } else {
        showError(message || '状态切换失败')
      }
    } catch (error) {
      showError('状态切换失败')
    }
    setStatusLoading(false)
  }

  const handleDelete = async() => {
    if (!window.confirm('确定要删除这个邀请码吗？')) return

    try {
      const res = await API.delete(`/api/invite-code/${item.id}`)
      const { success, message } = res.data
      if (success) {
        showSuccess('邀请码删除成功')
        onRefresh()
      } else {
        showError(message || '删除失败')
      }
    } catch (error) {
      showError('删除失败')
    }
  }

  const handleCopyCode = () => {
    copy(item.code, '邀请码')
  }

  const formatDate = (timestamp, defaultText = '未设置') => {
    if (!timestamp) return defaultText
    return timestamp2string(timestamp)
  }

  const getStatusLabel = (status) => {
    return status === STATUS.ENABLED ? (
      <Label color="success">启用</Label>
    ) : (
      <Label color="error">禁用</Label>
    )
  }

  return (
    <TableRow tabIndex={item.id}>
      <TableCell>
        <Checkbox
          checked={selected}
          onChange={onSelectRow}
        />
      </TableCell>
      <TableCell>{item.id}</TableCell>
      <TableCell>
        <Label
          variant="soft"
          color="default"
          sx={{
            fontFamily: 'monospace',
            fontSize: '0.875rem',
            cursor: 'pointer',
            '&:hover': {
              backgroundColor: 'primary.light',
              color: 'primary.main'
            }
          }}
          onClick={handleCopyCode}
        >
          {item.code}
        </Label>
      </TableCell>
      <TableCell>{item.name || '-'}</TableCell>
      <TableCell>
        {item.used_count} / {item.max_uses === 0 ? '∞' : item.max_uses}
      </TableCell>
      <TableCell>{formatDate(item.starts_at, '立即生效')}</TableCell>
      <TableCell>{formatDate(item.expires_at, '永不过期')}</TableCell>
      <TableCell>{formatDate(item.created_time)}</TableCell>
      <TableCell>
        <Tooltip title="点击切换状态">
          <TableSwitch
            id={`switch-${item.id}`}
            checked={statusSwitch === STATUS.ENABLED}
            onChange={handleStatus}
            disabled={statusLoading}
          />
        </Tooltip>
      </TableCell>
      <TableCell>
        <IconButton onClick={() => handleOpenModal(item)} sx={{ color: 'rgb(99, 115, 129)' }}>
          <Icon icon="solar:pen-bold-duotone"/>
        </IconButton>
        <IconButton onClick={handleDelete} sx={{ color: 'rgb(99, 115, 129)' }}>
          <Icon icon="solar:trash-bin-trash-bold-duotone"/>
        </IconButton>
      </TableCell>
    </TableRow>
  )
}

InviteCodeTableRow.propTypes = {
  item: PropTypes.object.isRequired,
  selected: PropTypes.bool.isRequired,
  onSelectRow: PropTypes.func.isRequired,
  onRefresh: PropTypes.func.isRequired,
  handleOpenModal: PropTypes.func.isRequired
}
