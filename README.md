# catalog_lab
## technologies: postgresql, js

## 1) events table:
- id(int, PK)
- name(string, <=100 chars)
- description(text)
- start_time(date, time)
- end_time(date, time)
- price(float)
- age limit(int)
- place(FK from places table)

## 2) places table:
- id(int, PK)
- name(string, <=100 chars)
- capacity(int)
- address(text)
- opening_date(date)
- area in m^2(float)

![Diagram](./images/Diagram.png)
