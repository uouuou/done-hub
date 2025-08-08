import { createSvgIcon } from '@mui/material/utils';

const LinuxdoProfileIcon = createSvgIcon(
  <svg width="24" height="24" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" fill="none"
    stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
    className="icon icon-tabler icons-tabler-outline icon-tabler-brand-linux">
    <clipPath id="a">
        <circle cx="12" cy="12" r="8.6" />
    </clipPath>
    <circle cx="12" cy="12" r="10" />
    <rect clipPath="url(#a)" x="2" y="2" width="20" height="6" />
    <rect clipPath="url(#a)" x="2" y="8" width="20" height="8" />
    <rect clipPath="url(#a)" x="2" y="16" width="20" height="6" />
  </svg>,
  'LINUX DO'
);

export default LinuxdoProfileIcon;