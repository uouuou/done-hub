import PropTypes from 'prop-types'
import { useTheme } from '@mui/material/styles'
import { Icon } from '@iconify/react'
import { FormControl, InputAdornment, InputLabel, MenuItem, OutlinedInput, Select, Stack } from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

// 设置dayjs为中文
dayjs.locale('zh-cn')

export default function InviteCodeTableToolBar({ filterName, handleFilterName, onSearch }) {
  const theme = useTheme()
  const grey500 = theme.palette.grey[500]

  // 处理回车键搜索
  const handleKeyDown = (event) => {
    if (event.key === 'Enter' && onSearch) {
      event.preventDefault()
      onSearch()
    }
  }

  return (
    <>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={{ xs: 3, sm: 2, md: 4 }}
        padding={'24px'}
        paddingBottom={'0px'}
        sx={{ width: '100%', '& > *': { flex: 1 } }}
      >
        <FormControl>
          <InputLabel htmlFor="invite-keyword-label">邀请码/名称</InputLabel>
          <OutlinedInput
            id="keyword"
            name="keyword"
            sx={{
              minWidth: '100%'
            }}
            label="邀请码/名称"
            value={filterName.keyword}
            onChange={handleFilterName}
            onKeyDown={handleKeyDown}
            placeholder="邀请码/名称"
            startAdornment={
              <InputAdornment position="start">
                <Icon icon="solar:ticket-bold-duotone" width={20} height={20} color={grey500}/>
              </InputAdornment>
            }
          />
        </FormControl>
      </Stack>

      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={{ xs: 3, sm: 2, md: 4 }}
        padding={'24px'}
        sx={{ width: '100%', '& > *': { flex: 1 } }}
      >
        <FormControl>
          <InputLabel htmlFor="invite-status-label">状态</InputLabel>
          <Select
            id="invite-status-label"
            label="状态"
            value={filterName.status}
            name="status"
            onChange={handleFilterName}
            sx={{
              minWidth: '100%'
            }}
            MenuProps={{
              PaperProps: {
                style: {
                  maxHeight: 200
                }
              }
            }}
          >
            <MenuItem key={0} value={0}>全部</MenuItem>
            <MenuItem key={1} value={1}>启用</MenuItem>
            <MenuItem key={2} value={2}>禁用</MenuItem>
          </Select>
        </FormControl>

        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="zh-cn">
          <FormControl>
            <DateTimePicker
              label="生效时间"
              name="starts_at_from"
              value={filterName.starts_at_from === 0 ? null : dayjs.unix(filterName.starts_at_from)}
              onChange={(value) => {
                if (value === null) {
                  handleFilterName({ target: { name: 'starts_at_from', value: 0 } })
                  return
                }
                const timestamp = value.unix()
                if (filterName.starts_at_to > 0 && timestamp > filterName.starts_at_to) {
                  handleFilterName({ target: { name: 'starts_at_to', value: 0 } })
                }
                handleFilterName({ target: { name: 'starts_at_from', value: timestamp } })
              }}
              slotProps={{
                textField: {
                  fullWidth: true,
                  variant: 'outlined'
                },
                actionBar: {
                  actions: ['clear', 'today', 'accept']
                }
              }}
            />
          </FormControl>
        </LocalizationProvider>

        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale="zh-cn">
          <FormControl>
            <DateTimePicker
              label="过期时间"
              name="starts_at_to"
              value={filterName.starts_at_to === 0 ? null : dayjs.unix(filterName.starts_at_to)}
              onChange={(value) => {
                if (value === null) {
                  handleFilterName({ target: { name: 'starts_at_to', value: 0 } })
                  return
                }
                const timestamp = value.unix()
                if (filterName.starts_at_from > 0 && timestamp <= filterName.starts_at_from) {
                  return
                }
                handleFilterName({ target: { name: 'starts_at_to', value: timestamp } })
              }}
              minDateTime={filterName.starts_at_from > 0 ? dayjs.unix(filterName.starts_at_from) : null}
              slotProps={{
                textField: {
                  fullWidth: true,
                  variant: 'outlined'
                },
                actionBar: {
                  actions: ['clear', 'today', 'accept']
                }
              }}
            />
          </FormControl>
        </LocalizationProvider>
      </Stack>
    </>
  )
}

InviteCodeTableToolBar.propTypes = {
  filterName: PropTypes.object,
  handleFilterName: PropTypes.func,
  onSearch: PropTypes.func
}
