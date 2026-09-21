import { createTheme } from '@mantine/core'

export const theme = createTheme({
  primaryColor: 'pink',
  defaultRadius: 'md',
  spacing: { xs: '8px', sm: '12px', md: '16px', lg: '24px', xl: '32px' },
  headings: { fontWeight: '600' },
  other: {
    zIndexHeader: 200,
    zIndexOverlay: 1000,
  },
})
