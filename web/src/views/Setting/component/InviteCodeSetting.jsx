import { useCallback, useEffect, useState } from 'react'
import { showError, showSuccess } from 'utils/common'
import AdminContainer from 'ui-component/AdminContainer'

import { useTheme } from '@mui/material/styles'
import Table from '@mui/material/Table'
import TableBody from '@mui/material/TableBody'
import TableContainer from '@mui/material/TableContainer'
import TablePagination from '@mui/material/TablePagination'
import LinearProgress from '@mui/material/LinearProgress'
import ButtonGroup from '@mui/material/ButtonGroup'
import Toolbar from '@mui/material/Toolbar'
import useMediaQuery from '@mui/material/useMediaQuery'
import Alert from '@mui/material/Alert'

import {
  Box,
  Button,
  Card,
  Checkbox,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormControlLabel,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography
} from '@mui/material'
import InviteCodeTableRow from './InviteCodeTableRow'
import KeywordTableHead from 'ui-component/TableHead'
import InviteCodeTableToolBar from './InviteCodeTableToolBar'
import ConfirmDialog from 'ui-component/confirm-dialog'
import { API } from 'utils/api'
import { useTranslation } from 'react-i18next'
import { getPageSize, PAGE_SIZE_OPTIONS, savePageSize } from 'constants'

import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import { Icon } from '@iconify/react'

// 常量定义 - 与后端保持一致
const INVITE_CODE_CONFIG = {
  SETTING_KEY: 'InviteCodeRegisterEnabled',
  STATUS: {
    ENABLED: 1,
    DISABLED: 2
  },
  VALIDATION: {
    MAX_BATCH_COUNT: 100,
    MIN_MAX_USES: 0, // 0表示无限使用
    CODE_LENGTH: 8,
    MAX_NAME_LENGTH: 100
  },
  API_ENDPOINTS: {
    LIST: '/api/invite-code/',
    STATISTICS: '/api/invite-code/statistics',
    GENERATE: '/api/invite-code/generate',
    BATCH_DELETE: '/api/invite-code/batch-delete'
  }
}

const InviteCodeSetting = () => {
  const { t } = useTranslation()
  const theme = useTheme()
  const matchUpMd = useMediaQuery(theme.breakpoints.up('sm'))

  // 列表状态
  const [page, setPage] = useState(0)
  const [order, setOrder] = useState('desc')
  const [orderBy, setOrderBy] = useState('id')
  const [rowsPerPage, setRowsPerPage] = useState(() => getPageSize('inviteCode'))
  const [listCount, setListCount] = useState(0)
  const [searching, setSearching] = useState(false)

  // 邀请码页面的状态管理（包含时间筛选）
  const originalKeyword = {
    keyword: '',
    status: 0,
    starts_at_from: 0,
    starts_at_to: 0
  }
  const [toolBarValue, setToolBarValue] = useState(originalKeyword)
  const [searchKeyword, setSearchKeyword] = useState(originalKeyword)

  const [inviteCodes, setInviteCodes] = useState([])
  const [refreshFlag, setRefreshFlag] = useState(false)

  // 多选状态
  const [selectedCodes, setSelectedCodes] = useState([])
  const [batchDeleteConfirm, setBatchDeleteConfirm] = useState(false)
  const [batchDeleting, setBatchDeleting] = useState(false)

  // 设置状态
  const [inviteCodeRegisterEnabled, setInviteCodeRegisterEnabled] = useState(false)
  const [settingLoading, setSettingLoading] = useState(false)

  // 对话框状态
  const [openDialog, setOpenDialog] = useState(false)
  const [editingCode, setEditingCode] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    code: '',
    max_uses: 0, // 默认无限制使用
    starts_at: null,
    expires_at: null,
    count: 1,
    status: INVITE_CODE_CONFIG.STATUS.ENABLED
  })

  // 排序处理
  const handleSort = (event, id) => {
    const isAsc = orderBy === id && order === 'asc'
    if (id !== '') {
      setOrder(isAsc ? 'desc' : 'asc')
      setOrderBy(id)
    }
  }

  // 分页处理
  const handleChangePage = (event, newPage) => {
    setPage(newPage)
  }

  const handleChangeRowsPerPage = (event) => {
    const newRowsPerPage = parseInt(event.target.value, 10)
    setPage(0)
    setRowsPerPage(newRowsPerPage)
    savePageSize('inviteCode', newRowsPerPage)
  }

  // 完全按照渠道页面的逻辑
  const handleToolBarValue = (event) => {
    setToolBarValue({ ...toolBarValue, [event.target.name]: event.target.value })
  }

  // 完全按照渠道页面的搜索逻辑
  const searchInviteCodes = async() => {
    // 如果正在搜索中，防止重复提交
    if (searching) {
      return
    }

    setPage(0)
    // 使用时间戳来确保即使搜索条件相同也能触发重新搜索
    const searchPayload = {
      ...toolBarValue,
      _timestamp: Date.now()
    }
    setSearchKeyword(searchPayload)
  }

  // 完全按照渠道页面的刷新逻辑
  const handleRefresh = async(reset) => {
    if (reset) {
      setOrderBy('id')
      setOrder('desc')
      setToolBarValue(originalKeyword)
      setSearchKeyword(originalKeyword)
    }
    setRefreshFlag(!refreshFlag)
  }

  // 完全按照渠道页面的数据获取逻辑
  const fetchData = useCallback(async(page, rowsPerPage, keyword, order, orderBy) => {
    setSearching(true)

    // 移除仅用于触发状态更新的时间戳字段
    if (keyword._timestamp) {
      delete keyword._timestamp
    }

    try {
      if (orderBy) {
        orderBy = order === 'desc' ? '-' + orderBy : orderBy
      }

      const params = {
        page: page + 1,
        size: rowsPerPage,
        order: orderBy,
        ...keyword  // 完全按照渠道页面的方式传递所有参数
      }

      const res = await API.get(INVITE_CODE_CONFIG.API_ENDPOINTS.LIST, { params })
      const { success, message, data } = res.data
      if (success) {
        setListCount(data.total_count)
        setInviteCodes(data.data || [])
      } else {
        showError(message)
      }
    } catch (error) {
      console.error(error)
      setInviteCodes([])
    }
    setSearching(false)
  }, [])

  // 多选处理
  const handleSelectAll = () => {
    if (selectedCodes.length === inviteCodes.length) {
      setSelectedCodes([])
    } else {
      setSelectedCodes(inviteCodes.map((code) => code.id))
    }
  }

  const handleSelectOne = (id) => {
    const selectedIndex = selectedCodes.indexOf(id)
    let newSelected = []

    if (selectedIndex === -1) {
      newSelected = newSelected.concat(selectedCodes, id)
    } else if (selectedIndex === 0) {
      newSelected = newSelected.concat(selectedCodes.slice(1))
    } else if (selectedIndex === selectedCodes.length - 1) {
      newSelected = newSelected.concat(selectedCodes.slice(0, -1))
    } else if (selectedIndex > 0) {
      newSelected = newSelected.concat(
        selectedCodes.slice(0, selectedIndex),
        selectedCodes.slice(selectedIndex + 1)
      )
    }

    setSelectedCodes(newSelected)
  }

  // 批量删除处理
  const handleBatchDelete = () => {
    if (selectedCodes.length === 0) {
      showError('请选择要删除的邀请码')
      return
    }
    setBatchDeleteConfirm(true)
  }

  const confirmBatchDelete = async() => {
    setBatchDeleting(true)
    try {
      const res = await API.post(INVITE_CODE_CONFIG.API_ENDPOINTS.BATCH_DELETE, {
        ids: selectedCodes
      })
      const { success, message } = res.data
      if (success) {
        showSuccess(`成功删除 ${selectedCodes.length} 个邀请码`)
        setSelectedCodes([])
        handleRefresh(false)
      } else {
        showError(message || '批量删除失败')
      }
    } catch (error) {
      showError('批量删除失败')
    }
    setBatchDeleting(false)
    setBatchDeleteConfirm(false)
  }

  // 加载设置
  const loadSettings = async() => {
    try {
      const res = await API.get('/api/option/')
      const { success, data } = res.data
      if (success) {
        const setting = data.find(item => item.key === INVITE_CODE_CONFIG.SETTING_KEY)
        setInviteCodeRegisterEnabled(setting?.value === 'true')
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  // 切换邀请码注册
  const toggleInviteCodeRegister = async() => {
    if (settingLoading) return

    setSettingLoading(true)
    try {
      const newValue = !inviteCodeRegisterEnabled

      // 如果要开启邀请码注册，先检查是否有有效的邀请码
      if (newValue) {
        const now = Math.floor(Date.now() / 1000)
        const hasValidCodes = Array.isArray(inviteCodes) && inviteCodes.some(code =>
            code.status === INVITE_CODE_CONFIG.STATUS.ENABLED &&
            (code.max_uses === 0 || code.used_count < code.max_uses)
          // &&
          // (code.starts_at === 0 || code.starts_at <= now) &&
          // (code.expires_at === 0 || code.expires_at > now)
        )

        if (!hasValidCodes) {
          showError('当前没有启用的邀请码，请先启用邀请码后再开启邀请码注册')
          setSettingLoading(false)
          return
        }
      }

      const res = await API.put('/api/option/', {
        key: INVITE_CODE_CONFIG.SETTING_KEY,
        value: newValue.toString()
      })
      const { success, message } = res.data
      if (success) {
        setInviteCodeRegisterEnabled(newValue)
        showSuccess(newValue ? '邀请码注册已开启' : '邀请码注册已关闭')
      } else {
        showError(message || '设置失败')
      }
    } catch (error) {
      showError('设置失败')
    }
    setSettingLoading(false)
  }

  // 对话框处理
  const handleOpenDialog = (code = null) => {
    if (code) {
      setEditingCode(code)
      setFormData({
        name: code.name,
        code: code.code,
        max_uses: code.max_uses,
        starts_at: code.starts_at ? dayjs(code.starts_at * 1000) : null,
        expires_at: code.expires_at ? dayjs(code.expires_at * 1000) : null,
        count: 1,
        status: code.status
      })
    } else {
      setEditingCode(null)
      setFormData({
        name: '',
        code: '',
        max_uses: INVITE_CODE_CONFIG.VALIDATION.MIN_MAX_USES,
        starts_at: null,
        expires_at: null,
        count: 1,
        status: INVITE_CODE_CONFIG.STATUS.ENABLED
      })
    }
    setOpenDialog(true)
  }

  const handleCloseDialog = () => {
    if (submitting) return
    setOpenDialog(false)
    setEditingCode(null)
    setSubmitting(false)
  }

  // 时间验证函数
  const getTimeValidationError = () => {
    if (formData.starts_at && formData.expires_at) {
      if (formData.starts_at.isAfter(formData.expires_at) || formData.starts_at.isSame(formData.expires_at)) {
        return '过期时间必须大于生效时间'
      }
    }
    return null
  }

  // 生成随机邀请码
  const generateRandomCode = async() => {
    if (generating) return

    setGenerating(true)
    try {
      const res = await API.get(INVITE_CODE_CONFIG.API_ENDPOINTS.GENERATE)
      const { success, data, message } = res.data
      if (success) {
        setFormData({ ...formData, code: data.code })
      } else {
        showError(message || '生成邀请码失败')
      }
    } catch (error) {
      showError('生成邀请码失败')
    } finally {
      setGenerating(false)
    }
  }

  // 表单验证
  const validateForm = () => {
    // 名称长度验证
    if (formData.name && formData.name.length > INVITE_CODE_CONFIG.VALIDATION.MAX_NAME_LENGTH) {
      showError(`名称长度不能超过${INVITE_CODE_CONFIG.VALIDATION.MAX_NAME_LENGTH}个字符`)
      return false
    }

    // 邀请码格式验证
    if (formData.code && formData.code.trim() !== formData.code) {
      showError('邀请码不能包含前后空格')
      return false
    }
    if (formData.code && !/^[a-zA-Z0-9_-]+$/.test(formData.code)) {
      showError('邀请码只能包含字母、数字、下划线和短横线')
      return false
    }

    // 邀请码长度验证
    if (formData.code && (formData.code.length < 3 || formData.code.length > 32)) {
      showError('邀请码长度必须在3-32个字符之间')
      return false
    }

    // 使用次数验证（0表示无限制使用）
    if (formData.max_uses < 0) {
      showError('最大使用次数不能小于0')
      return false
    }

    // 批量创建数量验证
    if (formData.count > INVITE_CODE_CONFIG.VALIDATION.MAX_BATCH_COUNT) {
      showError(`批量创建数量不能超过${INVITE_CODE_CONFIG.VALIDATION.MAX_BATCH_COUNT}个`)
      return false
    }

    // 时间验证
    if (formData.starts_at && formData.expires_at && (formData.starts_at.isAfter(formData.expires_at) || formData.starts_at.isSame(formData.expires_at))) {
      showError('过期时间必须大于生效时间')
      return false
    }

    return true
  }

  // 提交表单
  const handleSubmit = async() => {
    if (submitting) return

    // 表单验证
    if (!validateForm()) return

    setSubmitting(true)
    try {
      let res
      if (editingCode) {
        // 更新时不传code字段，因为邀请码不可修改
        const updateData = {
          name: formData.name,
          max_uses: formData.max_uses,
          status: formData.status,
          starts_at: formData.starts_at ? Math.floor(formData.starts_at.valueOf() / 1000) : 0,
          expires_at: formData.expires_at ? Math.floor(formData.expires_at.valueOf() / 1000) : 0
        }
        res = await API.put(`/api/invite-code/${editingCode.id}`, updateData)
      } else {
        // 创建时传所有字段
        const createData = {
          ...formData,
          starts_at: formData.starts_at ? Math.floor(formData.starts_at.valueOf() / 1000) : 0,
          expires_at: formData.expires_at ? Math.floor(formData.expires_at.valueOf() / 1000) : 0
        }
        res = await API.post('/api/invite-code/', createData)
      }

      const { success, message } = res.data
      if (success) {
        showSuccess(editingCode ? '邀请码更新成功' : '邀请码创建成功')
        handleCloseDialog()
        handleRefresh(false)
      } else {
        showError(message || '操作失败')
      }
    } catch (error) {
      showError('操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 效果钩子
  useEffect(() => {
    loadSettings()
  }, [])

  useEffect(() => {
    fetchData(page, rowsPerPage, searchKeyword, order, orderBy)
  }, [page, rowsPerPage, searchKeyword, order, orderBy, refreshFlag, fetchData])

  return (
    <AdminContainer>
      <Container maxWidth={false}>
        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="zh-cn">
          {/* 邀请码注册设置 */}
          <Card sx={{ mb: 3 }}>
            <Box sx={{ p: 2 }}>
              <Typography variant="h6" gutterBottom>
                邀请码注册设置
              </Typography>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={inviteCodeRegisterEnabled}
                    onChange={toggleInviteCodeRegister}
                    disabled={settingLoading}
                  />
                }
                label="启用邀请码注册"
              />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                启用后，用户注册时需要提供有效的邀请码
              </Typography>
            </Box>
          </Card>

          {/* 邀请码设置 */}
          <Stack direction="row" alignItems="center" justifyContent="space-between" mb={5}>
            <Stack direction="row" alignItems="center" spacing={1}>
              <Typography variant="h4">
                邀请码设置
              </Typography>
            </Stack>
            <ButtonGroup variant="contained" aria-label="outlined small primary button group">
              <Button
                color="primary"
                startIcon={<Icon icon="solar:add-circle-line-duotone"/>}
                onClick={() => handleOpenDialog()}
              >
                创建邀请码
              </Button>
            </ButtonGroup>
          </Stack>

          <Stack mb={5}>
            <Alert severity="info">
              邀请码用于控制用户注册，只有持有有效邀请码的用户才能注册账号
              <br/>
              邀请码可以设置使用次数限制和有效期限制
              <br/>
              邀请码状态：启用/禁用
              <br/>
              批量创建时将自动生成随机邀请码
            </Alert>
          </Stack>

          <Card>
            <Box component="form" noValidate>
              <InviteCodeTableToolBar
                filterName={toolBarValue}
                handleFilterName={handleToolBarValue}
                onSearch={searchInviteCodes}
              />
            </Box>

            {/* 按钮工具栏 - 完全按照渠道页面 */}
            <Toolbar
              sx={{
                textAlign: 'right',
                height: 50,
                display: 'flex',
                justifyContent: 'space-between',
                p: (theme) => theme.spacing(0, 1, 0, 3),
                minWidth: 0
              }}
            >
              {/* 左侧删除邀请码按钮 - 完全按照渠道页面 */}
              {matchUpMd && (
                <Button
                  variant="outlined"
                  onClick={handleBatchDelete}
                  disabled={selectedCodes.length === 0}
                  startIcon={<Icon icon="solar:trash-bin-2-bold-duotone" width={18}/>}
                  color="error"
                  sx={{
                    minWidth: 'auto',
                    whiteSpace: 'nowrap',
                    flexShrink: 0
                  }}
                >
                  删除邀请码 ({selectedCodes.length})
                </Button>
              )}

              <Box
                sx={{ flex: 1, overflow: 'hidden', minWidth: 0, display: 'flex', justifyContent: 'flex-end', ml: 2 }}>
                {matchUpMd ? (
                  <Box sx={{
                    overflow: 'auto',
                    maxWidth: '100%',
                    scrollBehavior: 'smooth',
                    '&::-webkit-scrollbar': {
                      height: '4px'
                    },
                    '&::-webkit-scrollbar-thumb': {
                      backgroundColor: 'rgba(0,0,0,0.2)',
                      borderRadius: '2px'
                    }
                  }}>
                    <ButtonGroup
                      variant="outlined"
                      aria-label="outlined small primary button group"
                      sx={{
                        flexWrap: 'nowrap',
                        minWidth: 'max-content',
                        display: 'flex'
                      }}
                    >
                      <Button
                        onClick={() => handleRefresh(true)}
                        startIcon={<Icon icon="solar:refresh-circle-bold-duotone" width={18}/>}
                        sx={{
                          whiteSpace: 'nowrap',
                          minWidth: 'auto',
                          px: 1.5
                        }}
                      >
                        刷新/清除搜索条件
                      </Button>
                      <Button
                        onClick={searchInviteCodes}
                        startIcon={
                          searching ? (
                            <Icon
                              icon="solar:refresh-bold-duotone"
                              width={18}
                              style={{
                                animation: 'spin 1s linear infinite',
                                color: '#1976d2'
                              }}
                            />
                          ) : (
                            <Icon icon="solar:magnifer-bold-duotone" width={18}/>
                          )
                        }
                        sx={{
                          whiteSpace: 'nowrap',
                          minWidth: 'auto',
                          px: 1.5,
                          ...(searching && {
                            bgcolor: 'action.hover',
                            color: 'primary.main',
                            '&:hover': {
                              bgcolor: 'action.selected'
                            }
                          })
                        }}
                      >
                        {searching ? '搜索中...' : '搜索'}
                      </Button>
                    </ButtonGroup>
                  </Box>
                ) : (
                  <Stack
                    direction="row"
                    spacing={0.5}
                    divider={<Divider orientation="vertical" flexItem/>}
                    justifyContent="space-around"
                    alignItems="center"
                    sx={{
                      overflow: 'auto',
                      minWidth: 'max-content',
                      '&::-webkit-scrollbar': {
                        height: '4px'
                      },
                      '&::-webkit-scrollbar-thumb': {
                        backgroundColor: 'rgba(0,0,0,0.2)',
                        borderRadius: '2px'
                      }
                    }}
                  >
                    <IconButton onClick={() => handleRefresh(true)} size="large">
                      <Icon width={20} icon="solar:refresh-circle-bold-duotone"/>
                    </IconButton>
                    <IconButton
                      onClick={searchInviteCodes}
                      size="large"
                      sx={{
                        ...(searching && {
                          bgcolor: 'action.hover',
                          color: 'primary.main'
                        })
                      }}
                    >
                      {searching ? (
                        <Icon
                          width={20}
                          icon="solar:refresh-bold-duotone"
                          style={{
                            animation: 'spin 1s linear infinite',
                            color: '#1976d2'
                          }}
                        />
                      ) : (
                        <Icon width={20} icon="solar:magnifer-bold-duotone"/>
                      )}
                    </IconButton>
                    <IconButton onClick={handleBatchDelete} disabled={selectedCodes.length === 0} size="large"
                                color="error">
                      <Icon width={20} icon="solar:trash-bin-2-bold-duotone"/>
                    </IconButton>
                  </Stack>
                )}
              </Box>
            </Toolbar>
            {searching && <LinearProgress/>}
            <TableContainer>
              <Table sx={{ minWidth: 800 }}>
                <KeywordTableHead
                  order={order}
                  orderBy={orderBy}
                  onRequestSort={handleSort}
                  numSelected={selectedCodes.length}
                  rowCount={inviteCodes.length}
                  onSelectAllClick={handleSelectAll}
                  headLabel={[
                    { id: 'select', label: '', disableSort: true, width: '50px' },
                    { id: 'id', label: 'ID', disableSort: false, width: '80px' },
                    { id: 'code', label: '邀请码', disableSort: false, width: '120px' },
                    { id: 'name', label: '名称', disableSort: false, width: '150px' },
                    { id: 'usage', label: '使用情况', disableSort: true, width: '100px' },
                    { id: 'starts_at', label: '生效时间', disableSort: false, width: '120px' },
                    { id: 'expires_at', label: '过期时间', disableSort: false, width: '120px' },
                    { id: 'created_time', label: '创建时间', disableSort: false, width: '120px' },
                    { id: 'status', label: '状态', disableSort: false, width: '80px' },
                    { id: 'action', label: '操作', disableSort: true, width: '100px' }
                  ]}
                />
                <TableBody>
                  {inviteCodes.map((row) => (
                    <InviteCodeTableRow
                      key={row.id}
                      item={row}
                      selected={selectedCodes.indexOf(row.id) !== -1}
                      onSelectRow={() => handleSelectOne(row.id)}
                      onRefresh={() => handleRefresh(false)}
                      handleOpenModal={handleOpenDialog}
                    />
                  ))}
                </TableBody>
              </Table>
            </TableContainer>

            <TablePagination
              page={page}
              component="div"
              count={listCount}
              rowsPerPage={rowsPerPage}
              onPageChange={handleChangePage}
              rowsPerPageOptions={PAGE_SIZE_OPTIONS}
              onRowsPerPageChange={handleChangeRowsPerPage}
              showFirstButton
              showLastButton
            />
          </Card>

          {/* 创建/编辑对话框 */}
          <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
            <DialogTitle
              sx={{ margin: '0px', fontWeight: 700, lineHeight: '1.55556', padding: '24px', fontSize: '1.125rem' }}>
              {editingCode ? '编辑邀请码' : '创建邀请码'}
            </DialogTitle>
            <Divider/>
            <DialogContent>
              <TextField
                fullWidth
                label="名称"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                margin="normal"
              />
              <TextField
                fullWidth
                label="邀请码"
                value={formData.code}
                onChange={(e) => {
                  const value = e.target.value
                  // 实时过滤特殊字符
                  const filteredValue = value.replace(/[^a-zA-Z0-9_-]/g, '')
                  setFormData({ ...formData, code: filteredValue })
                }}
                placeholder="留空自动生成或手动输入"
                disabled={formData.count > 1 || generating || editingCode}
                margin="normal"
                helperText={
                  editingCode
                    ? '编辑时邀请码不可修改'
                    : formData.count > 1
                      ? '批量创建时将自动生成邀请码'
                      : '只能包含字母、数字、下划线和短横线'
                }
                InputProps={{
                  endAdornment: !editingCode && (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={generateRandomCode}
                        disabled={formData.count > 1 || generating}
                        edge="end"
                        sx={{
                          color: 'primary.main',
                          '&:hover': {
                            backgroundColor: 'primary.light'
                          },
                          '&:disabled': {
                            color: 'action.disabled'
                          },
                          ...(generating && {
                            color: 'primary.dark'
                          })
                        }}
                      >
                        <Icon
                          icon="solar:refresh-bold-duotone"
                          style={{
                            ...(generating && {
                              animation: 'spin 1s linear infinite'
                            })
                          }}
                        />
                      </IconButton>
                    </InputAdornment>
                  )
                }}
              />
              <TextField
                fullWidth
                label="最大使用次数"
                type="number"
                value={formData.max_uses}
                onChange={(e) => setFormData({ ...formData, max_uses: parseInt(e.target.value) || 0 })}
                margin="normal"
                inputProps={{ min: 0 }}
                helperText="0为无限制使用"
              />
              {!editingCode && (
                <TextField
                  fullWidth
                  label="批量创建数量"
                  type="number"
                  value={formData.count}
                  onChange={(e) => setFormData({ ...formData, count: parseInt(e.target.value) || 1 })}
                  margin="normal"
                  inputProps={{ min: 1, max: 100 }}
                  helperText="一次最多创建100个邀请码"
                />
              )}
              <DateTimePicker
                label="生效时间"
                value={formData.starts_at}
                onChange={(newValue) => {
                  // 如果结束时间已设置且小于新的开始时间，则清空结束时间
                  if (formData.expires_at && newValue && formData.expires_at.isBefore(newValue)) {
                    setFormData({ ...formData, starts_at: newValue, expires_at: null })
                  } else {
                    setFormData({ ...formData, starts_at: newValue })
                  }
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                    margin: 'normal',
                    error: !!getTimeValidationError(),
                    helperText: getTimeValidationError() || '留空表示立即生效'
                  },
                  actionBar: {
                    actions: ['clear', 'today', 'accept']
                  }
                }}
              />
              <DateTimePicker
                label="过期时间"
                value={formData.expires_at}
                onChange={(newValue) => {
                  setFormData({ ...formData, expires_at: newValue })
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                    margin: 'normal',
                    error: !!getTimeValidationError(),
                    helperText: getTimeValidationError() || '留空表示永不过期'
                  },
                  actionBar: {
                    actions: ['clear', 'today', 'accept']
                  }
                }}
                minDateTime={formData.starts_at || dayjs()}
              />
              {editingCode && (
                <FormControl fullWidth margin="normal">
                  <InputLabel id="status-select-label">状态</InputLabel>
                  <Select
                    labelId="status-select-label"
                    label="状态"
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                  >
                    <MenuItem value={INVITE_CODE_CONFIG.STATUS.ENABLED}>启用</MenuItem>
                    <MenuItem value={INVITE_CODE_CONFIG.STATUS.DISABLED}>禁用</MenuItem>
                  </Select>
                </FormControl>
              )}
            </DialogContent>
            <DialogActions>
              <Button onClick={handleCloseDialog} disabled={submitting}>取消</Button>
              <Button
                onClick={handleSubmit}
                variant="contained"
                disabled={submitting}
                startIcon={submitting ? <Icon icon="solar:loading-line-duotone" className="animate-spin"/> : null}
              >
                {submitting ? (editingCode ? '更新中...' : '创建中...') : (editingCode ? '更新' : '创建')}
              </Button>
            </DialogActions>
          </Dialog>

          {/* 批量删除确认对话框 */}
          <ConfirmDialog
            open={batchDeleteConfirm}
            onClose={() => setBatchDeleteConfirm(false)}
            title={t('common.delete')}
            content={t('common.deleteConfirm', { title: `选中的 ${selectedCodes.length} 个邀请码` })}
            action={
              <Button
                variant="contained"
                color="error"
                onClick={confirmBatchDelete}
                disabled={batchDeleting}
                startIcon={batchDeleting ? <Icon icon="solar:loading-line-duotone" className="animate-spin"/> : null}
              >
                {batchDeleting ? '删除中...' : '删除'}
              </Button>
            }
          />
        </LocalizationProvider>
      </Container>
    </AdminContainer>
  )
}

export default InviteCodeSetting
