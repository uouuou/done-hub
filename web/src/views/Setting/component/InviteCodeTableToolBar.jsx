import PropTypes from 'prop-types'
import { useState } from 'react'
import {
  FormControl,
  IconButton,
  InputAdornment,
  InputLabel,
  MenuItem,
  OutlinedInput,
  Select,
  Stack
} from '@mui/material'
import { Icon } from '@iconify/react'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

// 常量定义 - 与主组件保持一致
const STATUS_OPTIONS = {
  ALL: 0,
  ENABLED: 1,
  DISABLED: 2
}

export default function InviteCodeTableToolBar({
  filterName,
  handleFilterName,
  onSearch,
  onSearchClick
}) {
  const [keyword, setKeyword] = useState('')

  const handleSubmit = (event) => {
    event.preventDefault()
    onSearch(keyword)
    if (onSearchClick) {
      onSearchClick()
    }
  }

  const handleKeywordChange = (event) => {
    setKeyword(event.target.value)
  }

  const handleKeyPress = (event) => {
    if (event.key === 'Enter') {
      handleSubmit(event)
    }
  }

  return (
    <>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={{ xs: 3, sm: 2, md: 4 }}
        padding={'24px'}
        sx={{ width: '100%', '& > *': { flex: 1 } }}
      >
        <FormControl>
          <OutlinedInput
            value={keyword}
            onChange={handleKeywordChange}
            onKeyPress={handleKeyPress}
            placeholder="搜索邀请码、名称"
            startAdornment={
              <InputAdornment position="start">
                <Icon icon="solar:magnifer-line-duotone"/>
              </InputAdornment>
            }
            endAdornment={
              <InputAdornment position="end">
                <IconButton onClick={handleSubmit} edge="end">
                  <Icon icon="solar:arrow-right-line-duotone"/>
                </IconButton>
              </InputAdornment>
            }
          />
        </FormControl>

        <FormControl>
          <InputLabel htmlFor="status-label">状态</InputLabel>
          <Select
            id="status-label"
            label="状态"
            value={filterName.status || STATUS_OPTIONS.ALL}
            name="status"
            onChange={handleFilterName}
            sx={{ minWidth: '100%' }}
            MenuProps={{
              PaperProps: {
                style: {
                  maxHeight: 200
                }
              }
            }}
          >
            <MenuItem value={STATUS_OPTIONS.ALL}>全部</MenuItem>
            <MenuItem value={STATUS_OPTIONS.ENABLED}>启用</MenuItem>
            <MenuItem value={STATUS_OPTIONS.DISABLED}>禁用</MenuItem>
          </Select>
        </FormControl>

        <FormControl>
          <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale={'zh-cn'}>
            <DateTimePicker
              label="生效开始时间"
              ampm={false}
              name="starts_at_from"
              value={filterName.starts_at_from === 0 ? null : dayjs.unix(filterName.starts_at_from)}
              onChange={(value) => {
                if (value === null) {
                  handleFilterName({ target: { name: 'starts_at_from', value: 0 } })
                  return
                }
                const timestamp = value.unix()
                // 如果结束时间已设置且小于开始时间，则清空结束时间
                if (filterName.starts_at_to > 0 && timestamp > filterName.starts_at_to) {
                  handleFilterName({ target: { name: 'starts_at_to', value: 0 } })
                }
                handleFilterName({ target: { name: 'starts_at_from', value: timestamp } })
              }}
              slotProps={{
                textField: {
                  fullWidth: true,
                  onKeyPress: (e) => e.preventDefault() // 禁用回车
                },
                actionBar: {
                  actions: ['clear', 'today', 'accept']
                }
              }}
            />
          </LocalizationProvider>
        </FormControl>

        <FormControl>
          <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale={'zh-cn'}>
            <DateTimePicker
              label="生效结束时间"
              name="starts_at_to"
              ampm={false}
              value={filterName.starts_at_to === 0 ? null : dayjs.unix(filterName.starts_at_to)}
              onChange={(value) => {
                if (value === null) {
                  handleFilterName({ target: { name: 'starts_at_to', value: 0 } })
                  return
                }
                const timestamp = value.unix()
                // 验证结束时间必须大于开始时间
                if (filterName.starts_at_from > 0 && timestamp <= filterName.starts_at_from) {
                  return // 不允许设置小于等于开始时间的结束时间
                }
                handleFilterName({ target: { name: 'starts_at_to', value: timestamp } })
              }}
              minDateTime={filterName.starts_at_from > 0 ? dayjs.unix(filterName.starts_at_from) : null}
              slotProps={{
                textField: {
                  fullWidth: true,
                  onKeyPress: (e) => e.preventDefault() // 禁用回车
                },
                actionBar: {
                  actions: ['clear', 'today', 'accept']
                }
              }}
            />
          </LocalizationProvider>
        </FormControl>
      </Stack>
    </>
  )
}

InviteCodeTableToolBar.propTypes = {
  filterName: PropTypes.object,
  handleFilterName: PropTypes.func,
  onSearch: PropTypes.func,
  onSearchClick: PropTypes.func
}
